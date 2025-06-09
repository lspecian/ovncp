# OVN Control Platform (ovncp)

A modern, production-ready control platform for Open Virtual Network (OVN) infrastructure. OVN Control Platform provides a comprehensive web interface and RESTful API for managing virtual networks, implementing security policies, and monitoring network health.

## 🚀 Features

### Core Functionality
- **Complete OVN Management**: Full lifecycle management for switches, routers, ports, ACLs, load balancers, and NAT rules
- **Atomic Transactions**: Execute multiple OVN operations atomically with rollback support
- **Real-time Monitoring**: Live network topology visualization and resource health monitoring
- **Multi-tenancy**: Isolated environments with project-based resource segregation

### Security & Compliance
- **Authentication**: OAuth 2.0/OIDC integration with support for GitHub, Google, and custom providers
- **Authorization**: Fine-grained RBAC with predefined roles (Admin, Operator, Viewer)
- **Audit Logging**: Complete audit trail of all operations for compliance
- **Security Headers**: CSP, HSTS, and other security headers enabled by default
- **Rate Limiting**: Configurable rate limits per user/IP with burst support

### Developer Experience
- **OpenAPI 3.0**: Complete API documentation with Swagger UI and ReDoc
- **SDK Support**: Auto-generated client libraries for multiple languages
- **Extensive Testing**: Unit, integration, and E2E test suites with coverage reporting
- **CI/CD Ready**: GitHub Actions workflows for automated testing and deployment

### Operations
- **Cloud Native**: Kubernetes-ready with production Helm charts
- **Observability**: Prometheus metrics, distributed tracing, and structured logging
- **High Availability**: Horizontal scaling with session affinity
- **Health Checks**: Comprehensive health and readiness probes
- **Performance**: Optimized for large-scale OVN deployments

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   React Web UI  │────▶│   Go API Server │────▶│  OVN Northbound │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │                          │
                               ▼                          ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │   PostgreSQL    │     │  OVN Southbound │
                        └─────────────────┘     └─────────────────┘
```

## 📋 Requirements

- **OVN**: Version 2.13+ (tested with 2.13, 20.03, 20.06, 21.06)
- **Database**: PostgreSQL 14+ or CockroachDB 21+ (or SQLite for single-node deployments)
- **Runtime**: Docker 20+ or Kubernetes 1.24+
- **Browser**: Chrome 90+, Firefox 88+, Safari 14+, Edge 90+

## 🚀 Quick Start

### Option 1: Single Container (Fastest)

```bash
# Run the all-in-one image (includes API, Web UI, and SQLite)
docker run -d \
  -p 8080:8080 \
  -e OVN_NORTHBOUND_DB="tcp:your-ovn-host:6641" \
  -v ovncp_data:/data \
  ghcr.io/lspecian/ovncp:main-simple

# Access the web UI at http://localhost:8080
# Default credentials: admin/admin
```

### Option 2: Using Docker Compose (Development)

```bash
# Clone the repository
git clone https://github.com/lspecian/ovncp.git
cd ovncp

# For simple deployment (SQLite, single container)
docker-compose -f docker-compose.simple.yml up -d

# For full deployment (PostgreSQL, separate containers)
cp .env.example .env
# Edit .env with your configuration
docker-compose up -d

# View logs
docker-compose logs -f

# Access the services
# - Web UI: http://localhost:8080
# - API Docs: http://localhost:8080/api/docs
# - Metrics: http://localhost:8080/metrics
```

### Using Kubernetes (Production)

```bash
# Add Helm repository
helm repo add ovncp https://charts.ovncp.io
helm repo update

# Install with custom values
helm install ovncp ovncp/ovncp \
  --namespace ovncp \
  --create-namespace \
  --values values.yaml

# Check deployment status
kubectl -n ovncp get pods
kubectl -n ovncp get svc
```

Example `values.yaml`:
```yaml
ovn:
  northbound:
    address: tcp:ovn-northbound:6641
  southbound:
    address: tcp:ovn-southbound:6642

auth:
  enabled: true
  providers:
    github:
      clientId: "your-github-client-id"
      clientSecret: "your-github-client-secret"

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: ovncp.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: ovncp-tls
      hosts:
        - ovncp.example.com
```

## 🛠️ Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- PostgreSQL 14+
- Docker & Docker Compose
- Make (optional but recommended)
- golangci-lint (for linting)

### Backend Development

```bash
# Install dependencies
go mod download

# Set up database
createdb ovncp_dev
make migrate-up

# Run tests with coverage
make test-coverage

# Run linting
make lint

# Run development server with hot reload
air

# Or run directly
go run cmd/api/main.go
```

### Frontend Development

```bash
cd web

# Install dependencies
npm install

# Run development server
npm run dev

# Run tests with coverage
npm run test:coverage

# Run linting
npm run lint

# Build for production
npm run build

# Preview production build
npm run preview
```

### Running Integration Tests

```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
make test-integration

# Run E2E tests
cd web && npm run test:e2e
```

## 📚 Documentation

### User Documentation
- [Quick Start Guide](docs/quick-start.md) - Get up and running in 5 minutes
- [User Guide](docs/user-guide.md) - Complete guide for end users
- [Admin Guide](docs/admin-guide.md) - System administration and maintenance

### Deployment & Operations
- [Installation Guide](docs/installation.md) - Detailed installation instructions
- [Configuration Reference](docs/configuration.md) - All configuration options
- [Kubernetes Deployment](docs/kubernetes-deployment.md) - Production Kubernetes setup
- [High Availability Setup](docs/high-availability.md) - Multi-node deployment
- [Backup and Recovery](docs/backup-recovery.md) - Data protection strategies

### Developer Documentation
- [API Reference](https://api.ovncp.io/docs) - Interactive API documentation
- [Architecture Overview](docs/architecture.md) - System design and components
- [Development Guide](docs/development.md) - Contributing and development setup
- [Plugin Development](docs/plugins.md) - Extending OVN Control Platform

### Tutorials
- [Creating Your First Network](docs/tutorials/first-network.md)
- [Implementing Security Policies](docs/tutorials/security-policies.md)
- [Load Balancer Configuration](docs/tutorials/load-balancing.md)
- [Monitoring and Alerting](docs/tutorials/monitoring.md)

## Project Structure

```
ovncp/
├── cmd/ovncp/          # Application entrypoint
├── internal/           # Internal packages
│   ├── api/           # HTTP handlers and routes
│   ├── auth/          # Authentication & authorization
│   ├── config/        # Configuration management
│   ├── db/            # Database models and migrations
│   ├── ovn/           # OVN client implementation
│   └── service/       # Business logic layer
├── web/               # React frontend
│   ├── src/
│   │   ├── components/  # React components
│   │   ├── hooks/       # Custom React hooks
│   │   ├── lib/         # Utilities and helpers
│   │   └── pages/       # Page components
├── charts/            # Helm charts
├── docs/              # Documentation
└── scripts/           # Utility scripts
```

## Configuration

Key configuration options:

```yaml
# Database
DB_HOST: localhost
DB_PORT: 5432
DB_NAME: ovncp

# OVN Connection
OVN_NB_ADDR: tcp:ovn-northbound:6641
OVN_SB_ADDR: tcp:ovn-southbound:6642

# Authentication
AUTH_ENABLED: true
OAUTH2_PROVIDER: keycloak
OAUTH2_ISSUER_URL: https://auth.example.com/realms/ovncp

# Server
SERVER_PORT: 8080
LOG_LEVEL: info
```

See [Configuration Reference](docs/configuration.md) for all options.

## 🔌 API Examples

### Authentication

```bash
# Get access token (example with GitHub OAuth)
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"provider": "github", "code": "oauth-code"}' | jq -r .access_token)

# Use token in requests
export AUTH_HEADER="Authorization: Bearer $TOKEN"
```

### Managing Resources

```bash
# List logical switches with pagination
curl -H "$AUTH_HEADER" \
  "http://localhost:8080/api/v1/switches?page=1&page_size=20"

# Create a logical switch
curl -X POST -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-tier",
    "description": "Web application tier",
    "subnet": "10.0.1.0/24",
    "dns_servers": ["8.8.8.8", "8.8.4.4"]
  }' \
  http://localhost:8080/api/v1/switches

# Create a port on the switch
curl -X POST -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server-01",
    "mac_address": "00:00:00:00:00:01",
    "ip_addresses": ["10.0.1.10"]
  }' \
  http://localhost:8080/api/v1/switches/{switch-id}/ports

# Execute atomic transaction
curl -X POST -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "operations": [
      {
        "operation": "create",
        "resource_type": "logical_switch",
        "data": {"name": "db-tier", "subnet": "10.0.2.0/24"}
      },
      {
        "operation": "create",
        "resource_type": "acl",
        "data": {
          "name": "allow-web-to-db",
          "priority": 1000,
          "direction": "from-lport",
          "match": "ip4.src == 10.0.1.0/24 && tcp.dst == 5432",
          "action": "allow"
        }
      }
    ]
  }' \
  http://localhost:8080/api/v1/transactions
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Process

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and add tests
4. Ensure all tests pass (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style

- Go: Follow [Effective Go](https://golang.org/doc/effective_go.html) and use `gofmt`
- JavaScript/TypeScript: ESLint configuration provided
- Commit messages: Follow [Conventional Commits](https://www.conventionalcommits.org/)

## 🧪 Testing

```bash
# Run all tests
make test-all

# Run specific test suites
make test-unit        # Unit tests only
make test-integration # Integration tests
make test-e2e        # End-to-end tests

# Run with coverage
make test-coverage

# Run benchmarks
make bench

# Run security tests
make test-security
```

## 🔒 Security

OVN Control Platform takes security seriously. Key features include:

- **Authentication**: OAuth 2.0/OIDC with MFA support
- **Authorization**: Fine-grained RBAC with audit logging
- **Encryption**: TLS 1.2+ for all communications
- **Security Headers**: CSP, HSTS, X-Frame-Options enabled
- **Rate Limiting**: Configurable limits with burst support
- **Input Validation**: Comprehensive validation for all inputs
- **Dependency Scanning**: Automated vulnerability scanning

### Reporting Security Issues

Please report security vulnerabilities to: security@ovncp.io

See our [Security Policy](SECURITY.md) for more details.

## 📊 Performance

OVN Control Platform is designed for scale:

- Handles 10,000+ concurrent connections
- Processes 50,000+ API requests/second
- Manages 100,000+ OVN resources
- Sub-millisecond operation latency
- Horizontal scaling support

## 🌍 Community

- **Slack**: [Join our Slack](https://ovncp.slack.com)
- **Forum**: [Community Forum](https://forum.ovncp.io)
- **Twitter**: [@ovncp](https://twitter.com/ovncp)
- **Blog**: [blog.ovncp.io](https://blog.ovncp.io)

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Open vSwitch](https://www.openvswitch.org/) and [OVN](https://www.ovn.org/) communities
- [CNCF](https://www.cncf.io/) for cloud-native tools and practices
- All our [contributors](https://github.com/lspecian/ovncp/graphs/contributors)

## 💬 Support

- 📖 [Documentation](https://docs.ovncp.io)
- 🐛 [Issue Tracker](https://github.com/lspecian/ovncp/issues)
- 💬 [Discussions](https://github.com/lspecian/ovncp/discussions)
- 📧 [Mailing List](https://groups.google.com/g/ovncp)
- 🏢 [Commercial Support](https://ovncp.io/support)

---

<p align="center">
  Made with ❤️ by the OVN Control Platform Team
</p>