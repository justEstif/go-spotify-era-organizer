# Self-Hosting Guide

This guide covers deploying Spotify Era Organizer on your own server.

## Prerequisites

- Linux server (Ubuntu 22.04+ recommended)
- Go 1.25.5+ (for building)
- PostgreSQL 16+
- Domain name (optional, for HTTPS)
- Spotify Developer account
- Last.fm API account (optional)

## Deployment Options

### Option 1: Docker Compose (Recommended)

The easiest way to deploy is using Docker Compose with the included configuration.

#### 1. Clone the Repository

```bash
git clone https://github.com/justEstif/go-spotify-era-organizer.git
cd go-spotify-era-organizer
```

#### 2. Create Environment File

```bash
cat > .env << 'EOF'
SPOTIFY_ID=your-spotify-client-id
SPOTIFY_SECRET=your-spotify-client-secret
LASTFM_API_KEY=your-lastfm-api-key
DATABASE_URL=postgres://spotify:spotify@postgres:5432/spotify_era?sslmode=disable
EOF
```

#### 3. Create Production Docker Compose

```bash
cat > compose.prod.yaml << 'EOF'
services:
  app:
    build: .
    container_name: spotify-era-app
    ports:
      - "8080:8080"
    environment:
      - SPOTIFY_ID=${SPOTIFY_ID}
      - SPOTIFY_SECRET=${SPOTIFY_SECRET}
      - LASTFM_API_KEY=${LASTFM_API_KEY}
      - DATABASE_URL=${DATABASE_URL}
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    container_name: spotify-era-db
    environment:
      POSTGRES_USER: spotify
      POSTGRES_PASSWORD: spotify
      POSTGRES_DB: spotify_era
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U spotify -d spotify_era"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  postgres_data:
EOF
```

#### 4. Create Dockerfile

```bash
cat > Dockerfile << 'EOF'
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o spotify-era-organizer ./cmd/spotify-era-organizer

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/spotify-era-organizer .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./spotify-era-organizer"]
EOF
```

#### 5. Deploy

```bash
# Build and start
docker compose -f compose.prod.yaml up -d --build

# Run migrations
docker compose -f compose.prod.yaml exec app \
  migrate -path migrations -database "$DATABASE_URL" up

# Check logs
docker compose -f compose.prod.yaml logs -f app
```

---

### Option 2: Binary Deployment

Deploy a pre-built binary directly on your server.

#### 1. Build the Binary

On your build machine:

```bash
# Clone and build
git clone https://github.com/justEstif/go-spotify-era-organizer.git
cd go-spotify-era-organizer

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o spotify-era-organizer ./cmd/spotify-era-organizer

# Package with migrations
tar -czvf spotify-era-organizer.tar.gz spotify-era-organizer migrations/
```

#### 2. Set Up PostgreSQL

On your server:

```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql << 'EOF'
CREATE USER spotify WITH PASSWORD 'your-secure-password';
CREATE DATABASE spotify_era OWNER spotify;
GRANT ALL PRIVILEGES ON DATABASE spotify_era TO spotify;
EOF
```

#### 3. Deploy the Binary

```bash
# Create app directory
sudo mkdir -p /opt/spotify-era-organizer
cd /opt/spotify-era-organizer

# Extract the package
sudo tar -xzvf /path/to/spotify-era-organizer.tar.gz

# Create environment file
sudo cat > /opt/spotify-era-organizer/.env << 'EOF'
SPOTIFY_ID=your-spotify-client-id
SPOTIFY_SECRET=your-spotify-client-secret
LASTFM_API_KEY=your-lastfm-api-key
DATABASE_URL=postgres://spotify:your-secure-password@localhost:5432/spotify_era?sslmode=disable
EOF
sudo chmod 600 /opt/spotify-era-organizer/.env
```

#### 4. Run Migrations

```bash
# Install migrate CLI
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Run migrations
source /opt/spotify-era-organizer/.env
migrate -path /opt/spotify-era-organizer/migrations -database "$DATABASE_URL" up
```

#### 5. Create Systemd Service

```bash
sudo cat > /etc/systemd/system/spotify-era-organizer.service << 'EOF'
[Unit]
Description=Spotify Era Organizer
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/spotify-era-organizer
EnvironmentFile=/opt/spotify-era-organizer/.env
ExecStart=/opt/spotify-era-organizer/spotify-era-organizer
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Start the service
sudo systemctl daemon-reload
sudo systemctl enable spotify-era-organizer
sudo systemctl start spotify-era-organizer

# Check status
sudo systemctl status spotify-era-organizer
sudo journalctl -u spotify-era-organizer -f
```

---

## Reverse Proxy Setup

For production, run behind a reverse proxy for HTTPS support.

### Nginx

```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Caddy

```caddyfile
your-domain.com {
    reverse_proxy localhost:8080
}
```

**Important:** Update your Spotify app's Redirect URI to `https://your-domain.com/callback`.

---

## Spotify App Configuration

### Development (localhost)

- Redirect URI: `http://127.0.0.1:8080/callback`

### Production (custom domain)

- Redirect URI: `https://your-domain.com/callback`

**Note:** You can add multiple redirect URIs to support both development and production.

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `SPOTIFY_ID` | Yes | Spotify app Client ID |
| `SPOTIFY_SECRET` | Yes | Spotify app Client Secret |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `LASTFM_API_KEY` | No | Last.fm API key for tag enrichment |

### Database URL Format

```
postgres://user:password@host:port/database?sslmode=disable
```

For remote databases with SSL:
```
postgres://user:password@host:port/database?sslmode=require
```

---

## Backup and Restore

### Backup Database

```bash
# Using pg_dump
pg_dump -U spotify -d spotify_era > backup.sql

# Compressed
pg_dump -U spotify -d spotify_era | gzip > backup.sql.gz
```

### Restore Database

```bash
# From SQL file
psql -U spotify -d spotify_era < backup.sql

# From compressed
gunzip -c backup.sql.gz | psql -U spotify -d spotify_era
```

---

## Monitoring

### Health Check

The application logs startup and request information. Check logs with:

```bash
# Systemd
sudo journalctl -u spotify-era-organizer -f

# Docker
docker compose logs -f app
```

### Database Connections

```bash
# Check active connections
psql -U spotify -d spotify_era -c "SELECT count(*) FROM pg_stat_activity WHERE datname = 'spotify_era';"
```

---

## Troubleshooting

### "Invalid state" error on login

This usually means:
1. The OAuth state expired (took more than 5 minutes to complete login)
2. Browser cookies are blocked
3. Using `localhost` instead of `127.0.0.1`

**Fix:** Use `http://127.0.0.1:8080` consistently.

### "Sync not available yet" error

The sync endpoint has a 1-hour cooldown. Check when the next sync is available via `GET /api/sync/status`.

### Database connection refused

1. Ensure PostgreSQL is running
2. Check the `DATABASE_URL` format
3. Verify the user has access to the database

### No tags fetched

If `LASTFM_API_KEY` is not set, tag fetching is disabled and era detection may not work well. Consider adding a Last.fm API key.

---

## Security Considerations

1. **Environment Variables**: Never commit `.env` files. Use secrets management in production.
2. **Database**: Use a strong password and restrict network access.
3. **HTTPS**: Always use HTTPS in production.
4. **Spotify Tokens**: OAuth tokens are stored in the database. Ensure database access is restricted.
5. **Session Cookies**: Sessions expire after 24 hours by default.
