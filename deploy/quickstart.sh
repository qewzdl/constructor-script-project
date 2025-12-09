#!/usr/bin/env bash
set -euo pipefail

if [[ ${1:-} == "--help" || ${1:-} == "-h" ]]; then
        cat <<'USAGE'
Usage: ./deploy/quickstart.sh [DOMAIN] [EMAIL]

Creates a production-ready configuration and launches the CMS with HTTPS
terminating reverse proxy and PostgreSQL via Docker Compose.

Arguments:
  DOMAIN    Your domain name (e.g., example.com) - required for SSL
  EMAIL     Admin email for Let's Encrypt notifications (optional)

If arguments are not provided, the script will prompt for them interactively.

This script generates secure credentials (database password, JWT secret, 
and setup access key) and starts the containers. After deployment, you'll 
receive a setup key to access the configuration wizard at:
  https://your-domain.com/setup?key=<your-key>

The script requires Docker with the Compose plugin available as `docker compose`.
USAGE
        exit 0
fi

PYTHON_BIN=$(command -v python3 || command -v python || true)
if [[ -z "$PYTHON_BIN" ]]; then
        echo "Python 3 is required to generate secure credentials." >&2
        exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
        echo "Docker is required to run this script." >&2
        exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
        echo "Docker Compose plugin (docker compose) is required." >&2
        exit 1
fi

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
ENV_FILE="$ROOT_DIR/deploy/.env.production"
DOT_ENV_FILE="$ROOT_DIR/deploy/.env"

existing_db_password=""
existing_jwt_secret=""

if [[ -f "$ENV_FILE" ]]; then
        echo "An existing deploy/.env.production file was found." >&2
        read -r -p "Overwrite it? [y/N] " answer
        case "$answer" in
                [Yy]*)
                        set -a
                        # shellcheck disable=SC1090
                        source "$ENV_FILE"
                        set +a
                        existing_db_password=${DB_PASSWORD:-}
                        existing_jwt_secret=${JWT_SECRET:-}
                        unset DB_PASSWORD JWT_SECRET
                        if [[ -n "$existing_db_password" ]]; then
                                echo "Reusing existing database password." >&2
                        fi
                        if [[ -n "$existing_jwt_secret" ]]; then
                                echo "Reusing existing JWT secret." >&2
                        fi
                        ;; 
                *) echo "Aborted."; exit 1;;
        esac
else
        existing_volume=$(docker volume ls --filter label=com.docker.compose.volume=postgres_data --format '{{.Name}}' | head -n1 || true)
        if [[ -n "$existing_volume" ]]; then
                cat <<'WARNING' >&2
Detected an existing PostgreSQL data volume but no deploy/.env.production credentials file.
To avoid authentication failures the database password must match the stored credentials.
WARNING
                read -r -p "Would you like to enter the existing database password? [y/N] " answer
                case "$answer" in
                        [Yy]*)
                                read -rs -p "Existing database password: " existing_db_password
                                echo >&2
                                if [[ -z "$existing_db_password" ]]; then
                                        echo "No password entered. Aborting to prevent mismatched credentials." >&2
                                        exit 1
                                fi
                                ;;
                        *)
                                cat >&2 <<INSTRUCTIONS
You can rerun the script after either providing the correct password or removing the
existing PostgreSQL volume (this will erase any stored data). For example:

  docker compose -f deploy/docker-compose.prod.yml down
  docker volume rm "${existing_volume}"

Afterwards rerun deploy/quickstart.sh to generate fresh credentials.
INSTRUCTIONS
                                exit 1
                                ;;
                esac
        fi
fi

if [[ -n "$existing_jwt_secret" && ${#existing_jwt_secret} -ge 32 ]]; then
        JWT_SECRET="$existing_jwt_secret"
else
        JWT_SECRET=$($PYTHON_BIN - <<'PY'
import secrets
print(secrets.token_urlsafe(48))
PY
)
fi

SETUP_KEY=$($PYTHON_BIN - <<'PY'
import secrets
print(secrets.token_urlsafe(32))
PY
)

if [[ -n "$existing_db_password" ]]; then
        DB_PASSWORD="$existing_db_password"
else
        DB_PASSWORD=$($PYTHON_BIN - <<'PY'
import secrets,string
alphabet = string.ascii_letters + string.digits
print(''.join(secrets.choice(alphabet) for _ in range(24)))
PY
)
fi

DB_USER="constructor"
DB_NAME="constructor"
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable"

# Get domain from argument or prompt
user_domain="${1:-}"
if [[ -z "$user_domain" ]]; then
	echo ""
	echo "Caddy requires a domain name to obtain SSL certificates."
	read -r -p "Enter your domain (e.g., example.com): " user_domain
fi

if [[ -z "$user_domain" ]]; then
	echo "Error: Domain is required for production deployment." >&2
	exit 1
fi

# Get email from argument or prompt
user_email="${2:-}"
if [[ -z "$user_email" ]]; then
	read -r -p "Enter admin email for Let's Encrypt notifications [admin@${user_domain}]: " user_email
fi

if [[ -z "$user_email" ]]; then
	user_email="admin@${user_domain}"
fi

cat > "$ENV_FILE" <<ENV
# Generated by deploy/quickstart.sh on $(date -Iseconds)
# Database credentials
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME
DB_SSLMODE=disable
DB_HOST=postgres
DB_PORT=5432
DATABASE_URL=$DATABASE_URL

# JWT authentication
JWT_SECRET=$JWT_SECRET

# Setup security (one-time access key)
SETUP_KEY=$SETUP_KEY

# Server configuration
ENVIRONMENT=production
PORT=8080
UPLOAD_DIR=./uploads

# Caddy reverse proxy (required for SSL)
SITE_DOMAIN=$user_domain
SITE_EMAIL=$user_email

# Optional features (can be enabled via environment or settings)
ENABLE_REDIS=false
ENABLE_EMAIL=false
ENABLE_METRICS=false

# Content settings below will be configured via web setup at /setup
# SITE_NAME=My Site
# SITE_DESCRIPTION=My site description
# SITE_URL=https://$user_domain
# CORS_ORIGINS=https://$user_domain

# Subtitle generation (uncomment to enable OpenAI Whisper auto-transcriptions)
# SUBTITLE_GENERATION_ENABLED=true
# SUBTITLE_PROVIDER=openai
# OPENAI_API_KEY=sk-your-openai-api-key
# OPENAI_MODEL=whisper-1
# SUBTITLE_PREFERRED_NAME=lesson-subtitles
# SUBTITLE_LANGUAGE=en
# SUBTITLE_PROMPT=
# SUBTITLE_TEMPERATURE=0.0
ENV

chmod 600 "$ENV_FILE"

# Provide a docker-compose compatible .env next to the compose file so that
# subsequent `docker compose` invocations continue using the generated
# credentials even when --env-file is omitted.
cp "$ENV_FILE" "$DOT_ENV_FILE"
chmod 600 "$DOT_ENV_FILE"

pushd "$ROOT_DIR/deploy" >/dev/null

docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build

popd >/dev/null

echo ""
echo "✓ Deployment finished successfully!"
echo ""
echo "Containers started:"
echo "  - PostgreSQL database"
echo "  - Constructor CMS API"
echo "  - Caddy reverse proxy with SSL for $user_domain"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔐 SETUP ACCESS KEY (save this securely!):"
echo ""
echo "    $SETUP_KEY"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo "  1. Ensure DNS: $user_domain → your server IP"
echo "  2. Wait 1-2 minutes for SSL certificate provisioning"
echo "  3. Visit: https://$user_domain/setup?key=$SETUP_KEY"
echo ""
echo "Complete the setup wizard to configure:"
echo "  • Site name, description, and URL"
echo "  • Administrator account"
echo "  • Languages and other preferences"
echo ""
echo "⚠️  Without this key, no one can access the setup page - keep it secure!"
echo ""
