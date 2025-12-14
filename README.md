# Constructor Script CMS

This repository contains the Constructor Script CMS backend written in Go.

## Quick production deployment

For a turnkey setup on a public server with automatic HTTPS certificates, see [Production Deployment Quickstart](docs/production-deployment.md).

To quickly verify the stack run:

```bash
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

This will use environment variables for database credentials and JWT secret. For a complete production setup, generate a dedicated
`deploy/.env.production` via `deploy/quickstart.sh` which will create secure credentials.

## Local development

- `make run` – run the API locally.
- `docker-compose up` – start the PostgreSQL + API stack defined for development. Credentials can be customized via `.env` file.

## Logging

The backend uses structured logging (zerolog). By default logs are written to stdout, but you can write logs to a file by setting the `LOG_FILE` environment variable. The file will be created (if it doesn't exist) and opened in append mode.

Example:

```bash
LOG_FILE=./logs/api.log make run
```

The application closes the log file on shutdown.

## Security headers

The backend allows same-origin framing by default and still sends restrictive defaults to prevent clickjacking from other origins. To embed the site in an iframe from additional hosts (for example inside an admin preview), set `CSP_FRAME_ANCESTORS` with a comma-separated list of allowed origins (e.g. `CSP_FRAME_ANCESTORS='self,http://localhost:8081'`). The middleware will mirror the same policy in the `Content-Security-Policy` header and adjust `X-Frame-Options` automatically.

## Automatic subtitle generation

The upload pipeline can generate WebVTT subtitles for videos using OpenAI Whisper. Provide an `OPENAI_API_KEY` (either through the environment or via **Settings → Site → Subtitles** in the admin panel) and the backend will enable the feature immediately. Detailed setup instructions are available in [docs/subtitle-generation.md](docs/subtitle-generation.md).
