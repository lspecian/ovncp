# Scaling Guide for OVN Control Platform

This guide covers horizontal scaling capabilities and deployment strategies for the OVN Control Platform.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Components](#components)
- [Deployment Strategies](#deployment-strategies)
- [Configuration](#configuration)
- [Load Balancing](#load-balancing)
- [Monitoring](#monitoring)
- [Best Practices](#best-practices)

## Overview

The OVN Control Platform supports horizontal scaling through:

- **Distributed coordination** using Redis
- **Session affinity** for WebSocket connections
- **Distributed locking** for synchronized operations
- **Cache synchronization** across nodes
- **Health check endpoints** for load balancers

## Architecture

### Multi-Node Architecture

```
                    ┌─────────────────┐
                    │  Load Balancer  │
                    │   (HAProxy/     │
                    │    Nginx/ALB)   │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌────▼─────┐        ┌────▼─────┐        ┌────▼─────┐
   │  Node 1  │        │  Node 2  │        │  Node 3  │
   │  ovncp   │        │  ovncp   │        │  ovncp   │
   └────┬─────┘        └────┬─────┘        └────┬─────┘
        │                    │                    │
        └────────────────────┼────────────────────┘
                             │
                    ┌────────▼────────┐
                    │      Redis      │
                    │   (Cluster/     │
                    │   Sentinel)     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   PostgreSQL    │
                    │   (Primary/     │
                    │    Replica)     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │       OVN       │
                    │  (NB/SB DBs)    │
                    └─────────────────┘
```

## Components

### 1. Cluster Coordinator

The cluster coordinator manages node discovery and health monitoring.

```go
// Initialize cluster coordinator
coordinator := cluster.NewCoordinator(&cluster.CoordinatorConfig{
    NodeID:            "node-1",
    Hostname:          hostname,
    IP:                "192.168.1.10",
    Port:              8080,
    HeartbeatInterval: 5 * time.Second,
    NodeTimeout:       30 * time.Second,
}, redisClient, logger)

// Start coordinator
coordinator.Start(ctx)

// Register event handlers
coordinator.RegisterEventHandler(cluster.EventNodeJoin, handleNodeJoin)
coordinator.RegisterEventHandler(cluster.EventNodeLeave, handleNodeLeave)
```

### 2. Session Affinity

WebSocket sessions are tied to specific nodes for connection stability.

```go
// Initialize session store
sessionStore := cluster.NewSessionStore(&cluster.SessionStoreConfig{
    NodeID:      "node-1",
    SessionTTL:  30 * time.Minute,
}, redisClient, logger)

// Register WebSocket session
sessionStore.Register(ctx, sessionID, userID)

// Check session location
nodeID, err := sessionStore.GetNodeForSession(ctx, sessionID)
```

### 3. Distributed Locking

Ensure exclusive access to resources across nodes.

```go
// Initialize lock manager
lockManager := cluster.NewLockManager(redisClient, logger)

// Acquire lock
lock, err := lockManager.AcquireLock(ctx, "switch:update:uuid", &cluster.LockOptions{
    TTL:        30 * time.Second,
    MaxRetries: 10,
})
defer lock.Release(ctx)

// Execute critical section
// ...
```

### 4. Cache Synchronization

Cache invalidation events are broadcast to all nodes.

```go
// Register cache invalidation handler
coordinator.RegisterEventHandler(cluster.EventCacheInvalidate, func(event *cluster.Event) {
    patterns := event.Data["patterns"].([]string)
    for _, pattern := range patterns {
        cache.Clear(ctx, pattern)
    }
})

// Publish cache invalidation
coordinator.PublishCacheInvalidation([]string{
    "switch:*",
    "topology:*",
})
```

## Deployment Strategies

### 1. Docker Compose (Development)

```yaml
version: '3.8'

services:
  ovncp-1:
    image: ghcr.io/lspecian/ovncp:latest
    environment:
      - NODE_ID=node-1
      - REDIS_URL=redis://redis:6379
      - DATABASE_URL=postgresql://user:pass@postgres:5432/ovncp
    depends_on:
      - redis
      - postgres

  ovncp-2:
    image: ghcr.io/lspecian/ovncp:latest
    environment:
      - NODE_ID=node-2
      - REDIS_URL=redis://redis:6379
      - DATABASE_URL=postgresql://user:pass@postgres:5432/ovncp
    depends_on:
      - redis
      - postgres

  ovncp-3:
    image: ghcr.io/lspecian/ovncp:latest
    environment:
      - NODE_ID=node-3
      - REDIS_URL=redis://redis:6379
      - DATABASE_URL=postgresql://user:pass@postgres:5432/ovncp
    depends_on:
      - redis
      - postgres

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - ovncp-1
      - ovncp-2
      - ovncp-3

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=ovncp
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
```

### 2. Kubernetes (Production)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ovncp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ovncp
  template:
    metadata:
      labels:
        app: ovncp
    spec:
      containers:
      - name: ovncp
        image: ghcr.io/lspecian/ovncp:latest
        env:
        - name: NODE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: NODE_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: REDIS_URL
          value: redis://redis-service:6379
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: ovncp-secrets
              key: database-url
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        startupProbe:
          httpGet:
            path: /health/startup
            port: 8080
          initialDelaySeconds: 0
          periodSeconds: 10
          failureThreshold: 30
---
apiVersion: v1
kind: Service
metadata:
  name: ovncp-service
spec:
  selector:
    app: ovncp
  ports:
  - port: 80
    targetPort: 8080
    name: http
  - port: 9090
    targetPort: 9090
    name: metrics
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ovncp-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ovncp
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Configuration

### Environment Variables

```bash
# Node identification
NODE_ID=node-1
NODE_HOSTNAME=ovncp-1.example.com
NODE_IP=192.168.1.10
NODE_PORT=8080

# Clustering
ENABLE_CLUSTERING=true
CLUSTER_HEARTBEAT_INTERVAL=5s
CLUSTER_NODE_TIMEOUT=30s

# Redis (for clustering)
REDIS_URL=redis://redis.example.com:6379
REDIS_CLUSTER_MODE=false
REDIS_SENTINEL_MASTER=mymaster
REDIS_SENTINEL_NODES=sentinel1:26379,sentinel2:26379,sentinel3:26379

# Session affinity
SESSION_STORE_TYPE=redis
SESSION_TTL=30m
SESSION_CLEANUP_INTERVAL=5m

# Connection pooling
OVN_MAX_CONNECTIONS=10
DB_MAX_OPEN_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
REDIS_POOL_SIZE=10

# Cache configuration
CACHE_TYPE=redis
CACHE_KEY_PREFIX=ovncp:node1:
```

### Configuration File

```yaml
# config.yaml
server:
  port: 8080
  metrics_port: 9090

cluster:
  enabled: true
  node_id: ${NODE_ID}
  node_ip: ${NODE_IP}
  heartbeat_interval: 5s
  node_timeout: 30s

redis:
  addresses:
    - redis-1:6379
    - redis-2:6379
    - redis-3:6379
  password: ${REDIS_PASSWORD}
  cluster_mode: true
  pool_size: 10

database:
  primary:
    host: postgres-primary
    port: 5432
    database: ovncp
    user: ${DB_USER}
    password: ${DB_PASSWORD}
  replicas:
    - host: postgres-replica-1
      port: 5432
    - host: postgres-replica-2
      port: 5432
  max_open_connections: 25
  max_idle_connections: 5
  connection_max_lifetime: 1h

ovn:
  northbound:
    endpoints:
      - tcp:ovn-nb-1:6641
      - tcp:ovn-nb-2:6641
      - tcp:ovn-nb-3:6641
  southbound:
    endpoints:
      - tcp:ovn-sb-1:6642
      - tcp:ovn-sb-2:6642
      - tcp:ovn-sb-3:6642
  max_connections: 10
  connection_timeout: 30s
```

## Load Balancing

### HAProxy Configuration

```haproxy
global
    maxconn 4096
    log stdout local0

defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms
    option httplog

frontend ovncp_frontend
    bind *:80
    bind *:443 ssl crt /etc/ssl/certs/ovncp.pem
    redirect scheme https if !{ ssl_fc }
    
    # Health check endpoint
    acl health_check path /health
    use_backend health_backend if health_check
    
    # WebSocket detection
    acl is_websocket hdr(Upgrade) -i WebSocket
    use_backend websocket_backend if is_websocket
    
    # Default backend
    default_backend api_backend

backend health_backend
    balance roundrobin
    option httpchk GET /health/live
    server node1 ovncp-1:8080 check
    server node2 ovncp-2:8080 check
    server node3 ovncp-3:8080 check

backend api_backend
    balance leastconn
    option httpchk GET /health/ready
    http-check expect status 200
    
    server node1 ovncp-1:8080 check weight 100
    server node2 ovncp-2:8080 check weight 100
    server node3 ovncp-3:8080 check weight 100

backend websocket_backend
    balance source  # Session affinity based on source IP
    option httpchk GET /health/ready
    http-check expect status 200
    
    # WebSocket specific options
    timeout tunnel 1h
    option http-server-close
    option forceclose
    
    server node1 ovncp-1:8080 check weight 100
    server node2 ovncp-2:8080 check weight 100
    server node3 ovncp-3:8080 check weight 100

listen stats
    bind *:8404
    stats enable
    stats uri /stats
    stats refresh 30s
```

### Nginx Configuration

```nginx
upstream ovncp_api {
    least_conn;
    server ovncp-1:8080 max_fails=3 fail_timeout=30s;
    server ovncp-2:8080 max_fails=3 fail_timeout=30s;
    server ovncp-3:8080 max_fails=3 fail_timeout=30s;
}

upstream ovncp_websocket {
    ip_hash;  # Session affinity
    server ovncp-1:8080 max_fails=3 fail_timeout=30s;
    server ovncp-2:8080 max_fails=3 fail_timeout=30s;
    server ovncp-3:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    listen 443 ssl http2;
    server_name ovncp.example.com;

    ssl_certificate /etc/ssl/certs/ovncp.crt;
    ssl_certificate_key /etc/ssl/private/ovncp.key;

    # Health checks
    location /health {
        proxy_pass http://ovncp_api;
        proxy_set_header Host $host;
        access_log off;
    }

    # WebSocket endpoints
    location /ws {
        proxy_pass http://ovncp_websocket;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket timeouts
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # API endpoints
    location / {
        proxy_pass http://ovncp_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Enable keepalive
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

## Monitoring

### Prometheus Metrics

Each node exposes metrics at `/metrics`:

```yaml
# Cluster metrics
ovncp_cluster_nodes_total{status="active"} 3
ovncp_cluster_node_info{node_id="node-1",hostname="ovncp-1",ip="192.168.1.10"} 1
ovncp_cluster_leader{node_id="node-1"} 1

# Session metrics
ovncp_sessions_total{node_id="node-1"} 150
ovncp_sessions_active{node_id="node-1"} 45
ovncp_session_migrations_total{from="node-1",to="node-2"} 5

# Lock metrics
ovncp_distributed_locks_acquired_total{node_id="node-1"} 1234
ovncp_distributed_locks_held{node_id="node-1"} 3
ovncp_distributed_lock_wait_seconds{quantile="0.99"} 0.15

# Cache synchronization metrics
ovncp_cache_invalidation_events_total{node_id="node-1"} 567
ovncp_cache_invalidation_latency_seconds{quantile="0.99"} 0.002
```

### Grafana Dashboard

Import the provided dashboard for cluster monitoring:

```json
{
  "dashboard": {
    "title": "OVN Control Platform - Cluster",
    "panels": [
      {
        "title": "Active Nodes",
        "targets": [
          {
            "expr": "count(ovncp_cluster_nodes_total{status='active'})"
          }
        ]
      },
      {
        "title": "Session Distribution",
        "targets": [
          {
            "expr": "ovncp_sessions_active"
          }
        ]
      },
      {
        "title": "Lock Contention",
        "targets": [
          {
            "expr": "rate(ovncp_distributed_lock_wait_seconds_sum[5m]) / rate(ovncp_distributed_lock_wait_seconds_count[5m])"
          }
        ]
      }
    ]
  }
}
```

## Best Practices

### 1. Node Configuration

- Use unique node IDs (UUIDs or pod names)
- Configure appropriate heartbeat intervals
- Set reasonable node timeout values
- Enable graceful shutdown handling

### 2. Session Management

- Implement session migration for maintenance
- Use session affinity for WebSocket connections
- Monitor session distribution across nodes
- Clean up stale sessions periodically

### 3. Lock Management

- Use appropriate lock TTLs
- Implement lock renewal for long operations
- Monitor lock contention metrics
- Release locks in defer blocks

### 4. Cache Strategy

- Use Redis for distributed caching
- Implement cache warming on startup
- Monitor cache hit rates
- Configure appropriate TTLs

### 5. Database Connections

- Use connection pooling
- Configure read replicas for queries
- Monitor connection pool metrics
- Implement circuit breakers

### 6. Health Checks

- Implement comprehensive health checks
- Use different endpoints for different purposes:
  - `/health/live` - Kubernetes liveness
  - `/health/ready` - Kubernetes readiness
  - `/health/startup` - Kubernetes startup
- Include dependency checks
- Return appropriate HTTP status codes

### 7. Monitoring and Alerting

- Monitor cluster health metrics
- Set up alerts for node failures
- Track session distribution
- Monitor lock contention
- Alert on cache synchronization failures

### 8. Deployment

- Use rolling updates
- Implement PodDisruptionBudgets
- Configure anti-affinity rules
- Use horizontal pod autoscaling
- Implement proper resource limits

## Troubleshooting

### Common Issues

1. **Node not joining cluster**
   - Check Redis connectivity
   - Verify node ID uniqueness
   - Check network connectivity
   - Review coordinator logs

2. **Session affinity not working**
   - Verify load balancer configuration
   - Check session store connectivity
   - Review session TTL settings
   - Monitor session migration events

3. **Lock contention**
   - Review lock TTL settings
   - Check for lock leaks
   - Monitor lock hold times
   - Consider operation batching

4. **Cache inconsistency**
   - Check Redis pub/sub connectivity
   - Monitor invalidation events
   - Review cache TTL settings
   - Check event handler registration

### Debug Endpoints

```bash
# Get cluster status
curl http://ovncp.example.com/api/v1/cluster/nodes

# Check specific node
curl http://ovncp.example.com/api/v1/cluster/nodes/node-1

# View sessions
curl http://ovncp.example.com/api/v1/cluster/sessions

# Check leader
curl http://ovncp.example.com/api/v1/cluster/leader
```

## Performance Tuning

### 1. Connection Pooling

```yaml
ovn:
  max_connections: 20
  connection_idle_timeout: 5m
  connection_lifetime: 30m

database:
  max_open_connections: 50
  max_idle_connections: 10
  connection_max_lifetime: 1h
```

### 2. Cache Configuration

```yaml
cache:
  redis:
    pool_size: 20
    max_retries: 3
    dial_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
```

### 3. Batch Processing

```yaml
batch_processor:
  batch_size: 100
  batch_timeout: 100ms
  max_concurrent: 4
```

## Security Considerations

### 1. Inter-node Communication

- Use TLS for Redis connections
- Encrypt cluster event data
- Implement node authentication
- Use network policies

### 2. Session Security

- Rotate session IDs periodically
- Implement session expiration
- Encrypt session data
- Monitor abnormal session patterns

### 3. Lock Security

- Use cryptographically secure lock values
- Implement lock ownership verification
- Monitor lock abuse
- Set maximum lock hold times