# OVNCP - Quick Start Guide

OVN Control Platform - A simple, homelab-friendly web UI for managing Open Virtual Network.

## 🚀 Quick Start (Recommended)

### Option 1: Docker (Simplest)

```bash
# Run the pre-built all-in-one image
docker run -d \
  -p 8080:8080 \
  -e OVN_NORTHBOUND_DB="tcp:your-ovn-host:6641" \
  -v ovncp_data:/data \
  ghcr.io/lspecian/ovncp:main-simple

# Or using docker-compose
git clone https://github.com/lspecian/ovncp.git
cd ovncp
docker-compose -f docker-compose.simple.yml up -d

# Access the web UI
open http://localhost:8080
```

Default login: `admin` / `admin`

### Option 2: Binary

```bash
# Build the binary
go build -o ovncp ./cmd/api

# Run with the quick-start script
./scripts/quick-start.sh

# Or run directly
DB_TYPE=sqlite ./ovncp
```

## 🔧 Configuration

### Minimal Configuration

For homelab use, you only need to set your OVN connection:

```bash
# Docker
docker run -d \
  -p 8080:8080 \
  -e OVN_NORTHBOUND_DB="tcp:192.168.1.10:6641" \
  -v ovncp_data:/data \
  ghcr.io/lspecian/ovncp:latest

# Binary
export OVN_NORTHBOUND_DB="tcp:192.168.1.10:6641"
./ovncp
```

### Database Options

- **SQLite** (default): Zero configuration, perfect for single-node
- **PostgreSQL**: Set `DB_TYPE=postgres` for multi-node deployments
- **Memory**: Set `DB_TYPE=memory` for testing

### Common Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_TYPE` | `sqlite` | Database type (sqlite, postgres, memory) |
| `DB_NAME` | `./data/ovncp.db` | Database path (SQLite) or name (PostgreSQL) |
| `OVN_NORTHBOUND_DB` | `tcp:127.0.0.1:6641` | OVN northbound connection |
| `AUTH_ENABLED` | `true` | Enable authentication |
| `API_PORT` | `8080` | Port to listen on |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

## 🏠 Homelab Deployment

### Running on the OVN host

If OVNCP runs on the same machine as OVN:

```bash
# Use Unix socket for better performance
export OVN_NORTHBOUND_DB="unix:/var/run/ovn/ovnnb_db.sock"
./ovncp
```

### Running remotely

```bash
# Connect to remote OVN
export OVN_NORTHBOUND_DB="tcp:ovn-host.local:6641"
./ovncp
```

## 📦 Data Storage

- SQLite database: `./data/ovncp.db` (or Docker volume)
- Auto-migrations on startup
- Easy backup: just copy the `.db` file

## 🔐 Security

- Default admin credentials: `admin`/`admin` (change after first login!)
- JWT-based authentication
- Support for OAuth providers (GitHub, Google, OIDC)

## 🚀 Production Deployment

When you're ready to scale:

1. Switch to PostgreSQL: `DB_TYPE=postgres`
2. Use the full docker-compose.yml
3. Configure OAuth providers
4. Enable HTTPS with a reverse proxy

## 📝 License

MIT License - see LICENSE file for details.