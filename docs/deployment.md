# OVN Control Platform Deployment Guide

This guide covers the deployment of OVN Control Platform (ovncp) using various methods including Docker, docker-compose, Kubernetes with Helm, and CI/CD pipelines.

## Prerequisites

- Docker 20.10+ (for container deployments)
- Kubernetes 1.26+ (for Kubernetes deployments)
- Helm 3.10+ (for Kubernetes deployments)
- PostgreSQL 14+ (external or containerized)
- OVN/OVS infrastructure accessible from deployment environment

## Configuration

### Environment Variables

The following environment variables configure the ovncp services:

#### API Service
```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=ovncp
DB_USER=ovncp
DB_PASSWORD=secure_password
DB_SSL_MODE=require

# OVN Configuration
OVN_NB_ADDR=tcp:ovn-northbound:6641
OVN_SB_ADDR=tcp:ovn-southbound:6642
OVN_REMOTE_CA=/certs/ovn-ca.crt
OVN_REMOTE_CERT=/certs/ovn-cert.crt
OVN_REMOTE_KEY=/certs/ovn-key.key

# Authentication
AUTH_ENABLED=true
OAUTH2_PROVIDER=keycloak
OAUTH2_CLIENT_ID=ovncp
OAUTH2_CLIENT_SECRET=client_secret
OAUTH2_ISSUER_URL=https://auth.example.com/realms/ovncp
OAUTH2_REDIRECT_URL=https://ovncp.example.com/auth/callback

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
LOG_LEVEL=info
LOG_FORMAT=json
```

#### Web Service
```bash
# API Configuration
VITE_API_URL=http://localhost:8080
VITE_APP_TITLE="OVN Control Platform"

# Feature Flags
VITE_ENABLE_DARK_MODE=true
VITE_ENABLE_TOPOLOGY_VIEW=true
```

## Deployment Methods

### 1. Docker Deployment

Build and run individual containers:

```bash
# Build images
docker build -t ovncp-api:latest .
docker build -t ovncp-web:latest ./web

# Run API container
docker run -d \
  --name ovncp-api \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e DB_PASSWORD=secure_password \
  -e OVN_NB_ADDR=tcp:ovn-northbound:6641 \
  -v $(pwd)/certs:/certs:ro \
  --network ovncp-network \
  ovncp-api:latest

# Run Web container
docker run -d \
  --name ovncp-web \
  -p 3000:80 \
  -e VITE_API_URL=http://ovncp-api:8080 \
  --network ovncp-network \
  ovncp-web:latest
```

### 2. Docker Compose Deployment

For local development:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

For production deployment:

```bash
# Use production compose file
docker-compose -f docker-compose.prod.yml up -d

# Scale API service
docker-compose -f docker-compose.prod.yml up -d --scale api=3
```

### 3. Kubernetes Deployment with Helm

#### Install from Chart

```bash
# Add Helm repository (if published)
helm repo add ovncp https://charts.ovncp.io
helm repo update

# Install with default values
helm install ovncp ovncp/ovncp \
  --namespace ovncp \
  --create-namespace

# Install with custom values
helm install ovncp ovncp/ovncp \
  --namespace ovncp \
  --create-namespace \
  -f custom-values.yaml
```

#### Install from Local Chart

```bash
# Install from local chart directory
helm install ovncp ./charts/ovncp \
  --namespace ovncp \
  --create-namespace

# Upgrade existing installation
helm upgrade ovncp ./charts/ovncp \
  --namespace ovncp \
  --reuse-values
```

#### Custom Values Example

Create a `custom-values.yaml` file:

```yaml
# API configuration
api:
  replicaCount: 3
  image:
    repository: ghcr.io/lspecian/ovncp
    tag: v1.0.0-api
  
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 200m
      memory: 256Mi
  
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70

# Web configuration
web:
  replicaCount: 2
  image:
    repository: ghcr.io/lspecian/ovncp
    tag: v1.0.0-web

# Database configuration
postgresql:
  enabled: true
  auth:
    postgresPassword: postgres
    username: ovncp
    password: secure_password
    database: ovncp

# Ingress configuration
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: ovncp.example.com
      paths:
        - path: /api
          pathType: Prefix
          service: api
        - path: /
          pathType: Prefix
          service: web
  tls:
    - secretName: ovncp-tls
      hosts:
        - ovncp.example.com

# OVN configuration
ovn:
  northbound:
    address: tcp:ovn-northbound.ovn-system.svc.cluster.local:6641
  southbound:
    address: tcp:ovn-southbound.ovn-system.svc.cluster.local:6642
  tls:
    enabled: true
    existingSecret: ovn-certs
```

#### Managing Secrets

Create secrets for sensitive data:

```bash
# Create database secret
kubectl create secret generic ovncp-db-secret \
  --namespace ovncp \
  --from-literal=username=ovncp \
  --from-literal=password=secure_password

# Create OAuth2 secret
kubectl create secret generic ovncp-oauth2-secret \
  --namespace ovncp \
  --from-literal=client-id=ovncp \
  --from-literal=client-secret=client_secret

# Create OVN certificates secret
kubectl create secret generic ovn-certs \
  --namespace ovncp \
  --from-file=ca.crt=ovn-ca.crt \
  --from-file=tls.crt=ovn-cert.crt \
  --from-file=tls.key=ovn-key.key
```

### 4. CI/CD with GitHub Actions

The repository includes a comprehensive CI/CD pipeline that:

1. **Tests**: Runs backend and frontend tests
2. **Builds**: Creates multi-architecture Docker images
3. **Publishes**: Pushes images to GitHub Container Registry
4. **Scans**: Performs security vulnerability scanning

#### Setting up CI/CD

1. Enable GitHub Actions in your repository
2. Configure repository secrets:
   - `GITHUB_TOKEN` (automatically provided)
   - Additional secrets for deployment targets

3. Push to trigger workflows:
   ```bash
   git push origin main  # Triggers full CI/CD
   git push origin feature/xyz  # Triggers tests only
   ```

#### Manual Image Build

Build and push images manually:

```bash
# Login to registry
docker login ghcr.io -u USERNAME

# Build and tag images
docker build -t ghcr.io/lspecian/ovncp:latest-api .
docker build -t ghcr.io/lspecian/ovncp:latest-web ./web

# Push images
docker push ghcr.io/lspecian/ovncp:latest-api
docker push ghcr.io/lspecian/ovncp:latest-web
```

## Database Management

### Migrations

Run database migrations:

```bash
# Using Docker
docker run --rm \
  -e DB_HOST=postgres \
  -e DB_NAME=ovncp \
  -e DB_USER=ovncp \
  -e DB_PASSWORD=secure_password \
  ghcr.io/lspecian/ovncp:latest-api \
  migrate up

# In Kubernetes
kubectl exec -it deployment/ovncp-api -n ovncp -- migrate up
```

### Backup and Restore

```bash
# Backup database
pg_dump -h localhost -U ovncp -d ovncp > ovncp-backup.sql

# Restore database
psql -h localhost -U ovncp -d ovncp < ovncp-backup.sql
```

## Monitoring and Operations

### Health Checks

The services expose health check endpoints:

- API: `http://localhost:8080/health`
- Web: `http://localhost:3000/health`

### Prometheus Metrics

The API service exposes Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

### Logging

Configure log levels and formats via environment variables:

```bash
LOG_LEVEL=debug  # debug, info, warn, error
LOG_FORMAT=text  # text, json
```

View logs:

```bash
# Docker
docker logs -f ovncp-api

# Kubernetes
kubectl logs -f deployment/ovncp-api -n ovncp

# Docker Compose
docker-compose logs -f api
```

### Troubleshooting

Common issues and solutions:

1. **Database Connection Failed**
   - Verify database credentials and connectivity
   - Check SSL/TLS requirements
   - Ensure database exists and migrations are run

2. **OVN Connection Failed**
   - Verify OVN northbound/southbound addresses
   - Check certificate paths and permissions
   - Ensure OVN services are accessible

3. **Authentication Issues**
   - Verify OAuth2 provider configuration
   - Check redirect URLs match configuration
   - Ensure client credentials are correct

4. **Performance Issues**
   - Enable autoscaling in Kubernetes
   - Increase resource limits
   - Check database query performance
   - Enable caching if available

## Security Considerations

1. **Network Security**
   - Use TLS for all external communications
   - Implement network policies in Kubernetes
   - Use private networks for internal services

2. **Secrets Management**
   - Never commit secrets to version control
   - Use Kubernetes secrets or external secret managers
   - Rotate credentials regularly

3. **Container Security**
   - Run containers as non-root users
   - Use minimal base images (Alpine)
   - Regularly update base images
   - Scan images for vulnerabilities

4. **Access Control**
   - Enable authentication for all endpoints
   - Implement RBAC for API access
   - Use OAuth2/OIDC for user authentication
   - Audit access logs regularly

## Upgrading

### Rolling Updates

For zero-downtime upgrades:

```bash
# Kubernetes with Helm
helm upgrade ovncp ./charts/ovncp \
  --namespace ovncp \
  --set api.image.tag=v1.1.0-api \
  --set web.image.tag=v1.1.0-web

# Docker Compose
docker-compose pull
docker-compose up -d --no-deps --scale api=2 api
docker-compose up -d --no-deps web
```

### Database Migrations

Always run migrations before upgrading:

```bash
# Backup database first
pg_dump -h localhost -U ovncp -d ovncp > backup-$(date +%Y%m%d).sql

# Run migrations
migrate up

# Then upgrade application
helm upgrade ovncp ./charts/ovncp --namespace ovncp
```

## Support

For issues and support:

1. Check the [troubleshooting section](#troubleshooting)
2. Review application logs
3. Open an issue on GitHub
4. Contact the development team

## License

This project is licensed under the MIT License. See LICENSE file for details.