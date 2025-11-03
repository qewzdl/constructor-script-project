# Production Deployment Quickstart

This guide explains how to launch the Constructor Script CMS on a public server with automatic HTTPS in just a few minutes.

## Prerequisites

- A server running Linux with ports **80** and **443** exposed to the internet.
- A DNS A/AAAA record that points your domain (e.g. `blog.example.com`) to the server.
- Docker Engine 20.10+ and the Docker Compose plugin (`docker compose`).
- An email address that can receive Let's Encrypt expiry notices.

## One-command setup

```bash
./deploy/quickstart.sh blog.example.com admin@example.com "My Blog"
```

The script performs the following:

1. Generates secure credentials for PostgreSQL and JWT signing.
2. Creates `deploy/.env.production` containing production-ready defaults.
3. Builds the CMS Docker image and starts three containers:
   - **postgres** – persistent PostgreSQL 15 database.
   - **api** – the Go-based CMS backend.
   - **caddy** – automatically obtains and renews HTTPS certificates via Let's Encrypt and proxies traffic to the API.
4. Persists data in Docker volumes (`postgres_data`, `uploads_data`, `caddy_data`, `caddy_config`).

Once the containers are up, the site becomes available at `https://<your-domain>` as soon as DNS resolves to the server.

## Customisation

- To change environment values, edit `deploy/.env.production` and re-run `docker compose --env-file deploy/.env.production -f deploy/docker-compose.prod.yml up -d`.
- Uploaded media is stored in the `uploads_data` volume. You can back it up with `docker run --rm -v constructor-script-project_uploads_data:/data busybox tar -czf - -C /data . > uploads.tgz`.
- Stop the stack with `docker compose --env-file deploy/.env.production -f deploy/docker-compose.prod.yml down`. Add `-v` to remove the volumes (this deletes the database and uploads).

## Troubleshooting

- Check container logs: `docker compose -f deploy/docker-compose.prod.yml logs -f`.
- Ensure ports 80/443 are open and DNS is correctly configured; Caddy cannot obtain certificates otherwise.
- Regenerate configuration by re-running the quickstart script. It will prompt before overwriting the existing `.env.production` file.
- If the script reports an existing PostgreSQL volume but no credentials file, supply the original database password when prompted or remove the volume before continuing. This prevents the API from rebooting due to password mismatches.
