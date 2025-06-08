# Docker Quick Start Guide

Get OVN Control Platform running quickly with Docker and docker-compose.

## Prerequisites

- Docker 20.10 or later
- Docker Compose 2.0 or later
- 4GB RAM minimum
- Access to OVN infrastructure

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/lspecian/ovncp.git
cd ovncp
```

### 2. Configure Environment

Create a `.env` file with your configuration:

```bash
cat > .env << EOF
# Database
POSTGRES_PASSWORD=changeme
DB_PASSWORD=changeme

# OVN Connection
OVN_NB_ADDR=tcp:192.168.1.10:6641
OVN_SB_ADDR=tcp:192.168.1.10:6642

# OAuth2 (optional, disable for testing)
AUTH_ENABLED=false
EOF
```

### 3. Start Services

```bash
# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f
```

### 4. Access the Application

Open your browser to http://localhost:3000

The API is available at http://localhost:8080

### 5. Initial Setup

If authentication is disabled, you can immediately access all features. If enabled, configure your OAuth2 provider first.

## Configuration Options

### Basic Configuration

Edit `.env` file for basic settings:

```bash
# API Configuration
SERVER_PORT=8080
LOG_LEVEL=info

# Web Configuration
VITE_API_URL=http://localhost:8080
VITE_APP_TITLE="My OVN Platform"

# Database
DB_HOST=postgres
DB_PORT=5432
DB_NAME=ovncp
DB_USER=ovncp
```

### Advanced Configuration

For production use, additional configuration:

```bash
# TLS for OVN
OVN_REMOTE_CA=/certs/ca.crt
OVN_REMOTE_CERT=/certs/client.crt
OVN_REMOTE_KEY=/certs/client.key

# OAuth2
AUTH_ENABLED=true
OAUTH2_PROVIDER=keycloak
OAUTH2_CLIENT_ID=ovncp
OAUTH2_CLIENT_SECRET=secret
OAUTH2_ISSUER_URL=https://auth.example.com/realms/ovncp
```

## Common Operations

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api
docker-compose logs -f web
```

### Restart Services

```bash
# Restart all
docker-compose restart

# Restart specific service
docker-compose restart api
```

### Update Images

```bash
# Pull latest images
docker-compose pull

# Recreate containers
docker-compose up -d
```

### Clean Up

```bash
# Stop and remove containers
docker-compose down

# Remove volumes (WARNING: deletes data)
docker-compose down -v
```

## Troubleshooting

### Cannot Connect to OVN

1. Check OVN addresses in `.env`
2. Ensure OVN is accessible from Docker network
3. Check firewall rules

### Database Connection Failed

1. Wait for PostgreSQL to be ready:
   ```bash
   docker-compose logs postgres
   ```
2. Check credentials match in `.env`

### Web UI Not Loading

1. Check API is running:
   ```bash
   curl http://localhost:8080/health
   ```
2. Check browser console for errors
3. Verify `VITE_API_URL` in `.env`

## Next Steps

- [Full Deployment Guide](deployment.md)
- [Kubernetes Deployment](kubernetes-deployment.md)
- [API Documentation](api-reference.md)
- [Configuration Reference](configuration.md)