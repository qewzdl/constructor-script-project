# Security Guidelines

## JWT Secret Generation

The application requires a JWT_SECRET environment variable for token signing. This secret is critical for application security:

### Production
Always provide a strong, unique JWT_SECRET:
```bash
# Generate a secure secret (min 32 characters recommended):
openssl rand -base64 48
```

Set via environment variable or in `deploy/.env.production`:
```
JWT_SECRET=<your-generated-secret>
```

### Development
For local development, use the provided `.env` file with a development secret. The application logs a warning if JWT_SECRET is not provided or auto-generated:
```
WARNING: JWT_SECRET environment variable not set. Auto-generating a secure secret. For production, set JWT_SECRET to a strong, unique value.
```

## Database Credentials

### Production
Always generate unique credentials:
```bash
# Generate secure password (min 24 characters):
openssl rand -base64 32 | tr '+/' '-_' | cut -c1-24
```

Use the quickstart script to generate production configuration:
```bash
./deploy/quickstart.sh yourdomain.com admin@example.com "Site Name"
```

### Development
Development credentials are specified in `.env`:
- DB_USER: devuser
- DB_PASSWORD: devpassword
- DB_NAME: constructor

**Never commit production credentials to version control.**

## Environment-Specific Security

### Production
- All credentials (DB_PASSWORD, JWT_SECRET) must be strong and unique
- Use `deploy/.env.production` (excluded from git)
- Credentials should be managed by your infrastructure/secrets management system
- Never log or expose credentials

### Development
- Development credentials are non-sensitive and provided in `.env`
- For local development, credentials in `.env` are sufficient
- To test with custom credentials, update `.env` and restart containers

## Automatic Secret Generation

If JWT_SECRET is missing or too short (<32 chars):
1. Application auto-generates a secure random secret
2. A warning is logged with the cause
3. Database still starts successfully (for development)
4. **Always set JWT_SECRET explicitly in production**

## Related Files
- `.env` - Development configuration (committed to git)
- `.env.example` - Example configuration template
- `deploy/.env.production.example` - Production configuration template
- `deploy/.env.production` - Production configuration (NOT committed)
- `Dockerfile` - Reads environment variables
