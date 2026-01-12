# Wantok Deployment Guide

This guide covers deploying Wantok to a private VPS using Docker, with automated CI/CD via GitHub Actions.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         VPS                                  │
│  ┌─────────────┐     ┌─────────────┐     ┌──────────────┐  │
│  │   Caddy     │────▶│   Wantok    │────▶│   SQLite     │  │
│  │ (HTTPS/443) │     │  (:8080)    │     │   (data/)    │  │
│  └─────────────┘     └─────────────┘     └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
         ▲
         │ HTTPS
         │
    [Users/Browsers]
```

## Prerequisites

### On your VPS:
- Docker and Docker Compose installed
- SSH access configured
- Domain name pointing to VPS IP (for HTTPS)
- Ports 80 and 443 open in firewall

### On GitHub:
- Repository secrets configured (see below)
- Docker Hub account

## Initial VPS Setup

### 1. Install Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in for group changes to take effect
```

### 2. Create Application Directory

```bash
sudo mkdir -p /opt/wantok
sudo chown $USER:$USER /opt/wantok
cd /opt/wantok
```

### 3. Create Configuration Files

```bash
# Create .env file
cat > .env << 'EOF'
DOCKER_IMAGE=yourusername/wantok:latest
SESSION_SECRET=$(openssl rand -hex 32)
DOMAIN=wantok.yourdomain.com
EOF

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  wantok:
    image: ${DOCKER_IMAGE:-wantok:latest}
    container_name: wantok
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - wantok-data:/app/data
    environment:
      - DATABASE_PATH=/app/data/wantok.db
      - PORT=8080
      - SESSION_SECRET=${SESSION_SECRET}
      - SECURE_COOKIES=true
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/login"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s

  caddy:
    image: caddy:2-alpine
    container_name: wantok-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config
    depends_on:
      - wantok

volumes:
  wantok-data:
  caddy-data:
  caddy-config:
EOF

# Create Caddyfile (replace domain)
cat > Caddyfile << 'EOF'
wantok.yourdomain.com {
    reverse_proxy wantok:8080
    encode gzip

    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        -Server
    }
}
EOF
```

### 4. Generate SSH Key for GitHub Actions

```bash
# On your local machine or VPS
ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_actions

# Add public key to VPS authorized_keys
cat ~/.ssh/github_actions.pub >> ~/.ssh/authorized_keys

# Copy private key for GitHub secrets (you'll need this later)
cat ~/.ssh/github_actions
```

## GitHub Repository Setup

### 1. Configure Repository Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions

Add these secrets:

| Secret Name | Description | Example |
|-------------|-------------|---------|
| `DOCKERHUB_USERNAME` | Your Docker Hub username | `myusername` |
| `DOCKERHUB_TOKEN` | Docker Hub access token | `dckr_pat_xxx...` |
| `VPS_HOST` | Your VPS IP or hostname | `123.45.67.89` |
| `VPS_USERNAME` | SSH username on VPS | `deploy` |
| `VPS_SSH_KEY` | Private SSH key (from step 4) | `-----BEGIN OPENSSH...` |

### 2. Create Docker Hub Access Token

1. Go to [Docker Hub](https://hub.docker.com) → Account Settings → Security
2. Click "New Access Token"
3. Name it "GitHub Actions" and copy the token
4. Add as `DOCKERHUB_TOKEN` secret in GitHub

## Deployment

### First Deployment

```bash
# On VPS
cd /opt/wantok

# Pull and start services
docker compose pull
docker compose up -d

# Check logs
docker compose logs -f
```

### Automated Deployments

After initial setup, pushing to `main` branch triggers:

1. **Build** - GitHub Actions builds Docker image
2. **Push** - Image pushed to Docker Hub with `latest` and SHA tags
3. **Deploy** - SSH to VPS, pull new image, restart services

### Manual Deployment

```bash
# On VPS
cd /opt/wantok
docker compose pull
docker compose down
docker compose up -d
```

## User Management

### Create Admin User

After first deployment, create an admin user:

```bash
# On VPS
docker compose exec wantok /app/wantok --create-admin

# Follow prompts:
# username: admin
# password: (enter secure password)
# display name: Administrator
```

### Managing Users

1. Log in as admin at `https://yourdomain.com`
2. Click "Admin" in the header
3. From the admin panel you can:
   - Create new users
   - Edit user display names
   - Reset passwords
   - Toggle admin status
   - Delete users

### User Password Requirements

- Minimum 8 characters
- Maximum 128 characters

### Username Requirements

- 3-32 characters
- Letters, numbers, and underscores only

## Database Management

### Backup

Database is stored in Docker volume. To backup:

```bash
# Create backup directory
mkdir -p /opt/wantok/backups

# Backup database
docker compose exec wantok sqlite3 /app/data/wantok.db ".backup '/app/data/backup.db'"
docker cp wantok:/app/data/backup.db /opt/wantok/backups/wantok_$(date +%Y%m%d).db
docker compose exec wantok rm /app/data/backup.db
```

### Automated Backups

Add to crontab (`crontab -e`):

```cron
# Daily backup at 2 AM
0 2 * * * cd /opt/wantok && docker compose exec -T wantok sqlite3 /app/data/wantok.db ".backup '/app/data/backup.db'" && docker cp wantok:/app/data/backup.db /opt/wantok/backups/wantok_$(date +\%Y\%m\%d).db && find /opt/wantok/backups -name "wantok_*.db" -mtime +7 -delete
```

### Restore from Backup

```bash
# Stop services
docker compose down

# Copy backup to volume
docker run --rm -v wantok_wantok-data:/data -v /opt/wantok/backups:/backup alpine cp /backup/wantok_YYYYMMDD.db /data/wantok.db

# Start services
docker compose up -d
```

## Monitoring

### View Logs

```bash
# All services
docker compose logs -f

# Just Wantok
docker compose logs -f wantok

# Just Caddy
docker compose logs -f caddy
```

### Health Check

```bash
# Check service status
docker compose ps

# Check health
docker inspect wantok --format='{{.State.Health.Status}}'
```

### Resource Usage

```bash
docker stats wantok wantok-caddy
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker compose logs wantok

# Common issues:
# - SESSION_SECRET not set in .env
# - Port 8080 already in use
# - Volume permissions
```

### HTTPS Not Working

```bash
# Check Caddy logs
docker compose logs caddy

# Common issues:
# - Domain not pointing to VPS
# - Ports 80/443 blocked by firewall
# - Caddyfile syntax error
```

### Database Issues

```bash
# Check database file
docker compose exec wantok ls -la /app/data/

# Check database integrity
docker compose exec wantok sqlite3 /app/data/wantok.db "PRAGMA integrity_check;"
```

### Reset Everything

```bash
# WARNING: This deletes all data
docker compose down -v
docker compose up -d
# Then create admin user again
```

## Security Checklist

- [ ] Strong `SESSION_SECRET` (32+ random characters)
- [ ] HTTPS enabled via Caddy
- [ ] SSH key authentication only (disable password auth)
- [ ] Firewall configured (only 80, 443, 22 open)
- [ ] Regular backups configured
- [ ] Docker images kept updated

## Updating

### Update Wantok

Push to `main` branch triggers automatic deployment, or manually:

```bash
cd /opt/wantok
docker compose pull
docker compose down
docker compose up -d
```

### Update Caddy

```bash
docker compose pull caddy
docker compose up -d caddy
```
