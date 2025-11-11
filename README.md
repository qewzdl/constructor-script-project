# Constructor Script CMS

This repository contains the Constructor Script CMS backend written in Go.

## Quick production deployment

For a turnkey setup on a public server with automatic HTTPS certificates, see [Production Deployment Quickstart](docs/production-deployment.md).

To quickly verify the stack with the bundled defaults run:

```bash
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

This uses the bundled defaults (`bloguser`/`blogpassword`, a demo JWT secret and
`https://localhost`). For public deployments generate a dedicated
`deploy/.env.production` via `deploy/quickstart.sh`.

## Local development

- `make run` – run the API locally.
- `docker-compose up` – start the PostgreSQL + API stack defined for development.

## Automatic subtitle generation

The upload pipeline can generate WebVTT subtitles for videos using OpenAI Whisper. Provide an `OPENAI_API_KEY` (either through the environment or via **Settings → Site → Subtitles** in the admin panel) and the backend will enable the feature immediately. Detailed setup instructions are available in [docs/subtitle-generation.md](docs/subtitle-generation.md).
