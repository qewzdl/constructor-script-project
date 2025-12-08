# Production Deployment Quickstart

This guide explains how to launch the Constructor Script CMS on a public server with automatic HTTPS in just a few minutes.

## Prerequisites

- A server running Linux with ports **80** and **443** exposed to the internet.
- A DNS A/AAAA record that points your domain (e.g. `site.example.com`) to the server.
- Docker Engine 20.10+ and the Docker Compose plugin (`docker compose`).
- An email address that can receive Let's Encrypt expiry notices.

## One-command setup

```bash
./deploy/quickstart.sh site.example.com admin@example.com "My Site"
```

The script performs the following:

1. Generates secure credentials for PostgreSQL and JWT signing.
2. Creates `deploy/.env.production` containing production-ready defaults.
3. Builds the CMS Docker image and starts three containers:
   - **postgres** – persistent PostgreSQL 15 database.
   - **api** – the Go-based CMS backend.
   - **caddy** – automatically obtains and renews HTTPS certificates via Let's Encrypt and proxies traffic to the API.
4. Writes a `deploy/.env` file mirroring `deploy/.env.production` so that future `docker compose` commands automatically use the generated credentials.
5. Persists data in Docker volumes (`postgres_data`, `uploads_data`, `caddy_data`, `caddy_config`).

Once the containers are up, the site becomes available at `https://<your-domain>` as soon as DNS resolves to the server.

## Required environment variables

The following environment variables **must** be set before deploying to production:

- `DB_USER` – PostgreSQL username (must be provided, no default)
- `DB_PASSWORD` – PostgreSQL password (must be provided, no default)
- `JWT_SECRET` – Secret key for JWT token signing (must be provided, no default, minimum 32 characters recommended)
- `SITE_DOMAIN` – Your domain name (e.g., `site.example.com`)
- `SITE_EMAIL` – Email for Let's Encrypt certificate notifications

Generate these securely using:
```bash
openssl rand -base64 32  # for DB_PASSWORD
openssl rand -base64 48  # for JWT_SECRET
```

Never hardcode secrets or reuse default values. Always use the `./deploy/quickstart.sh` script which generates secure credentials automatically.

### Quick smoke-test without configuration

For staging or verification you can also run the stack without generating an
environment file. The compose definition will use environment variables from the system or `.env` file, so the following
command will launch the CMS on `https://localhost`:

```bash
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

**Important:** You must provide secure database credentials and JWT secret via environment variables or `.env` file before running in production. Never use default credentials on public servers.

## Customisation

- To change environment values, create or edit `deploy/.env.production` and re-run `docker compose --env-file deploy/.env.production -f deploy/docker-compose.prod.yml up -d`.
- Uploaded media is stored in the `uploads_data` volume. You can back it up with `docker run --rm -v constructor-script-project_uploads_data:/data busybox tar -czf - -C /data . > uploads.tgz`.
- Stop the stack with `docker compose --env-file deploy/.env.production -f deploy/docker-compose.prod.yml down`. Add `-v` to remove the volumes (this deletes the database and uploads).

## Troubleshooting

- Check container logs: `docker compose -f deploy/docker-compose.prod.yml logs -f`.
- Ensure ports 80/443 are open and DNS is correctly configured; Caddy cannot obtain certificates otherwise.
- Regenerate configuration by re-running the quickstart script. It will prompt before overwriting the existing `.env.production` file.
- If the script reports an existing PostgreSQL volume but no credentials file, supply the original database password when prompted or remove the volume before continuing. This prevents the API from rebooting due to password mismatches.
