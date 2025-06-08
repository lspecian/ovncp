# Kubernetes Deployment Guide

Deploy OVN Control Platform on Kubernetes using Helm.

## Prerequisites

- Kubernetes cluster 1.26+
- Helm 3.10+
- kubectl configured with cluster access
- Ingress controller (optional)
- cert-manager (optional, for TLS)

## Installation

### Quick Install

```bash
# Install with default values
helm install ovncp ./charts/ovncp \
  --namespace ovncp \
  --create-namespace
```

### Production Install

1. **Create namespace and secrets:**

```bash
# Create namespace
kubectl create namespace ovncp

# Database secret
kubectl create secret generic ovncp-postgresql \
  --namespace ovncp \
  --from-literal=postgres-password=adminpass \
  --from-literal=password=userpass

# OAuth2 secret
kubectl create secret generic ovncp-oauth2 \
  --namespace ovncp \
  --from-literal=client-id=ovncp \
  --from-literal=client-secret=your-secret

# OVN certificates
kubectl create secret generic ovn-certs \
  --namespace ovncp \
  --from-file=ca.crt=path/to/ca.crt \
  --from-file=tls.crt=path/to/client.crt \
  --from-file=tls.key=path/to/client.key
```

2. **Create values file `production-values.yaml`:**

```yaml
# Production configuration
global:
  storageClass: "fast-ssd"

api:
  replicaCount: 3
  
  image:
    repository: ghcr.io/lspecian/ovncp
    tag: v1.0.0-api
    pullPolicy: IfNotPresent
  
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 200m
      memory: 256Mi
  
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
  
  podDisruptionBudget:
    minAvailable: 2
  
  env:
    - name: LOG_LEVEL
      value: "info"
    - name: LOG_FORMAT
      value: "json"

web:
  replicaCount: 2
  
  image:
    repository: ghcr.io/lspecian/ovncp
    tag: v1.0.0-web
  
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi

postgresql:
  enabled: true
  architecture: replication
  auth:
    existingSecret: ovncp-postgresql
    database: ovncp
    username: ovncp
  primary:
    persistence:
      enabled: true
      size: 20Gi
  readReplicas:
    replicaCount: 2
    persistence:
      enabled: true
      size: 20Gi

redis:
  enabled: true
  architecture: replication
  auth:
    enabled: true
    password: "redis-password"
  master:
    persistence:
      enabled: true
      size: 8Gi

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
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

ovn:
  northbound:
    address: tcp:ovn-northbound.ovn-system:6641
  southbound:
    address: tcp:ovn-southbound.ovn-system:6642
  tls:
    enabled: true
    existingSecret: ovn-certs

oauth2:
  enabled: true
  provider: keycloak
  existingSecret: ovncp-oauth2
  issuerUrl: https://auth.example.com/realms/ovncp
  redirectUrl: https://ovncp.example.com/auth/callback

monitoring:
  serviceMonitor:
    enabled: true
  prometheusRule:
    enabled: true

networkPolicy:
  enabled: true
```

3. **Install with production values:**

```bash
helm install ovncp ./charts/ovncp \
  --namespace ovncp \
  -f production-values.yaml
```

## Configuration

### Storage

Configure persistent storage:

```yaml
postgresql:
  persistence:
    enabled: true
    storageClass: "fast-ssd"
    size: 50Gi

redis:
  master:
    persistence:
      enabled: true
      storageClass: "fast-ssd"
      size: 10Gi
```

### High Availability

Enable HA features:

```yaml
api:
  replicaCount: 3
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - ovncp
              - key: app.kubernetes.io/component
                operator: In
                values:
                  - api
          topologyKey: kubernetes.io/hostname

podDisruptionBudget:
  enabled: true
  minAvailable: 2
```

### Network Policies

Enable network isolation:

```yaml
networkPolicy:
  enabled: true
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              name: ovn-system
      ports:
        - protocol: TCP
          port: 6641
        - protocol: TCP
          port: 6642
```

### Resource Management

Set appropriate resource limits:

```yaml
api:
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi
    requests:
      cpu: 500m
      memory: 512Mi
  
  # Vertical Pod Autoscaler
  vpa:
    enabled: true
    updateMode: "Auto"
```

## Operations

### Monitoring

1. **Prometheus ServiceMonitor:**

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    path: /metrics
    labels:
      prometheus: kube-prometheus
```

2. **Grafana Dashboard:**

Import the dashboard from `charts/ovncp/dashboards/ovncp-dashboard.json`

3. **Alerts:**

```yaml
monitoring:
  prometheusRule:
    enabled: true
    rules:
      - alert: OVNCPAPIDown
        expr: up{job="ovncp-api"} == 0
        for: 5m
        annotations:
          summary: "OVNCP API is down"
```

### Backup and Restore

1. **Database Backup:**

```bash
# Create backup
kubectl exec -n ovncp postgresql-primary-0 -- \
  pg_dump -U ovncp ovncp | gzip > ovncp-backup-$(date +%Y%m%d).sql.gz

# Restore backup
gunzip -c ovncp-backup-20240106.sql.gz | \
  kubectl exec -i -n ovncp postgresql-primary-0 -- \
  psql -U ovncp ovncp
```

2. **Velero Backup:**

```yaml
apiVersion: velero.io/v1
kind: BackupStorageLocation
metadata:
  name: ovncp-backup
spec:
  provider: aws
  config:
    region: us-east-1
  objectStorage:
    bucket: ovncp-backups
```

### Scaling

1. **Manual Scaling:**

```bash
# Scale API deployment
kubectl scale deployment ovncp-api -n ovncp --replicas=5

# Scale using Helm
helm upgrade ovncp ./charts/ovncp \
  --namespace ovncp \
  --reuse-values \
  --set api.replicaCount=5
```

2. **Autoscaling:**

Already configured in values:

```yaml
api:
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
```

### Upgrading

1. **Check changes:**

```bash
helm diff upgrade ovncp ./charts/ovncp \
  --namespace ovncp \
  -f production-values.yaml
```

2. **Perform upgrade:**

```bash
# Upgrade with new chart version
helm upgrade ovncp ./charts/ovncp \
  --namespace ovncp \
  -f production-values.yaml

# Upgrade with new image
helm upgrade ovncp ./charts/ovncp \
  --namespace ovncp \
  --reuse-values \
  --set api.image.tag=v1.1.0-api \
  --set web.image.tag=v1.1.0-web
```

3. **Rollback if needed:**

```bash
# Check history
helm history ovncp -n ovncp

# Rollback to previous version
helm rollback ovncp -n ovncp
```

## Troubleshooting

### Pod Issues

```bash
# Check pod status
kubectl get pods -n ovncp

# Check pod logs
kubectl logs -n ovncp deployment/ovncp-api

# Describe pod for events
kubectl describe pod -n ovncp ovncp-api-xxx

# Execute into pod
kubectl exec -it -n ovncp deployment/ovncp-api -- /bin/sh
```

### Connectivity Issues

```bash
# Test API connectivity
kubectl run -it --rm debug \
  --image=nicolaka/netshoot \
  --restart=Never \
  -n ovncp -- curl http://ovncp-api:8080/health

# Test OVN connectivity
kubectl exec -it -n ovncp deployment/ovncp-api -- \
  nc -zv ovn-northbound.ovn-system 6641
```

### Resource Issues

```bash
# Check resource usage
kubectl top pods -n ovncp

# Check HPA status
kubectl get hpa -n ovncp

# Check PVC status
kubectl get pvc -n ovncp
```

## Security

### Pod Security Standards

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ovncp
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### RBAC

The Helm chart creates appropriate RBAC roles. For additional access:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ovncp-reader
  namespace: ovncp
rules:
  - apiGroups: [""]
    resources: ["pods", "services"]
    verbs: ["get", "list", "watch"]
```

### Secret Rotation

```bash
# Rotate database password
kubectl create secret generic ovncp-postgresql-new \
  --namespace ovncp \
  --from-literal=password=newpass \
  --dry-run=client -o yaml | kubectl apply -f -

# Update deployment
helm upgrade ovncp ./charts/ovncp \
  --namespace ovncp \
  --reuse-values \
  --set postgresql.auth.existingSecret=ovncp-postgresql-new
```

## Maintenance

### Health Checks

```bash
# Check API health
kubectl exec -n ovncp deployment/ovncp-api -- wget -qO- http://localhost:8080/health

# Check all endpoints
for pod in $(kubectl get pods -n ovncp -l app.kubernetes.io/name=ovncp -o name); do
  echo "Checking $pod"
  kubectl exec -n ovncp $pod -c api -- wget -qO- http://localhost:8080/health
done
```

### Log Collection

```bash
# Stream logs
kubectl logs -f -n ovncp -l app.kubernetes.io/name=ovncp

# Get logs from all pods
kubectl logs -n ovncp -l app.kubernetes.io/name=ovncp --all-containers=true

# Export logs
kubectl logs -n ovncp deployment/ovncp-api --since=1h > api-logs.txt
```

## Additional Resources

- [Helm Chart Reference](../charts/ovncp/README.md)
- [Configuration Reference](configuration.md)
- [API Documentation](api-reference.md)
- [Troubleshooting Guide](troubleshooting.md)