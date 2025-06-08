# OVN Control Platform Monitoring Stack

This directory contains the complete monitoring and observability stack for OVN Control Platform.

## Components

### Metrics
- **Prometheus**: Time-series database for metrics collection
- **Grafana**: Visualization and dashboards
- **Alertmanager**: Alert routing and notification

### Tracing
- **Jaeger**: Distributed tracing backend
- **OpenTelemetry**: Tracing instrumentation

### Logging
- **Loki**: Log aggregation system
- **Promtail**: Log collector
- **Structured Logging**: JSON-formatted logs with trace correlation

## Quick Start

### 1. Start the Monitoring Stack

```bash
# Start all monitoring services
docker-compose -f docker-compose.monitoring.yml up -d

# Verify services are running
docker-compose -f docker-compose.monitoring.yml ps
```

### 2. Access the Services

- **Grafana**: http://localhost:3001 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Alertmanager**: http://localhost:9093
- **Jaeger**: http://localhost:16686

### 3. Configure OVNCP Services

Update your OVNCP configuration to enable monitoring:

```yaml
# API configuration
METRICS_ENABLED: true
TRACING_ENABLED: true
TRACING_ENDPOINT: http://jaeger:4317
TRACING_SERVICE_NAME: ovncp-api
LOG_FORMAT: json
LOG_LEVEL: info
```

## Metrics

### Available Metrics

#### HTTP Metrics
- `ovncp_http_requests_total`: Total HTTP requests by method, endpoint, and status
- `ovncp_http_request_duration_seconds`: Request latency histogram
- `ovncp_http_request_size_bytes`: Request size histogram
- `ovncp_http_response_size_bytes`: Response size histogram

#### OVN Operation Metrics
- `ovncp_ovn_operations_total`: Total OVN operations by type and status
- `ovncp_ovn_operation_duration_seconds`: OVN operation latency
- `ovncp_ovn_connection_status`: OVN connection health (1=connected, 0=disconnected)

#### Business Metrics
- `ovncp_logical_switches_total`: Total number of logical switches
- `ovncp_logical_routers_total`: Total number of logical routers
- `ovncp_logical_ports_total`: Total number of logical ports
- `ovncp_acl_rules_total`: Total number of ACL rules
- `ovncp_load_balancers_total`: Total number of load balancers

#### System Metrics
- `ovncp_goroutines_count`: Current number of goroutines
- `ovncp_memory_usage_bytes`: Current memory usage
- `ovncp_db_connections_active`: Active database connections
- `ovncp_db_connections_idle`: Idle database connections

### Custom Queries

#### Request Rate
```promql
sum(rate(ovncp_http_requests_total[5m])) by (method)
```

#### P95 Latency
```promql
histogram_quantile(0.95, sum(rate(ovncp_http_request_duration_seconds_bucket[5m])) by (endpoint, le))
```

#### Error Rate
```promql
sum(rate(ovncp_http_requests_total{status=~"5.."}[5m])) / sum(rate(ovncp_http_requests_total[5m]))
```

## Dashboards

### Pre-configured Dashboards

1. **OVN Control Platform - Overview**
   - Request rate and latency
   - Resource counts
   - Error rates
   - System health

2. **OVN Control Platform - Performance**
   - Detailed latency percentiles
   - Throughput metrics
   - Resource utilization
   - Cache performance

### Importing Dashboards

1. Open Grafana at http://localhost:3001
2. Navigate to Dashboards → Import
3. Upload JSON files from `grafana/dashboards/`

## Alerts

### Alert Rules

Alerts are defined in `prometheus/alerts/ovncp-alerts.yml`:

#### Availability Alerts
- `OVNCPAPIDown`: API service is down
- `OVNConnectionLost`: Lost connection to OVN
- `DatabaseConnectionPoolExhausted`: No idle DB connections

#### Performance Alerts
- `HighAPILatency`: P95 latency > 1s
- `VeryHighAPILatency`: P95 latency > 5s
- `HighOVNOperationLatency`: OVN operations slow

#### Error Alerts
- `HighErrorRate`: Error rate > 5%
- `VeryHighErrorRate`: Error rate > 10%
- `PanicsDetected`: Application panics

#### SLO Alerts
- `SLOAvailabilityBreach`: < 99.9% availability
- `SLOLatencyBreach`: P95 > 500ms
- `SLOErrorRateBreach`: Error rate > 1%

### Alert Routing

Alerts are routed based on severity and component:
- **Critical**: PagerDuty + Slack + Email
- **Warning**: Slack + Email
- **Info**: Email only

Configure alert destinations in `alertmanager/alertmanager.yml`.

## Distributed Tracing

### Trace Propagation

Traces are automatically propagated through:
- HTTP headers (W3C Trace Context)
- All OVN operations
- Database queries
- External service calls

### Finding Traces

1. Open Jaeger UI at http://localhost:16686
2. Select service: `ovncp-api`
3. Search by:
   - Operation name
   - Tags (user_id, resource_id, etc.)
   - Duration
   - Error status

### Trace Analysis

Look for:
- Slow operations (high duration)
- Failed spans (error=true)
- High span counts (potential N+1 queries)
- Missing instrumentation gaps

## Logging

### Log Aggregation

Logs are collected from:
- Container stdout/stderr
- Application JSON logs
- System logs

### Log Queries in Grafana

1. Add Loki data source in Grafana
2. Use LogQL queries:

```logql
# All errors
{job="ovncp-api"} |= "error"

# Specific user
{job="ovncp-api"} | json | user_id="12345"

# Slow requests
{job="ovncp-api"} | json | duration_seconds > 1
```

### Structured Log Fields

- `timestamp`: ISO8601 timestamp
- `level`: Log level (debug, info, warn, error)
- `message`: Log message
- `trace_id`: Correlation with traces
- `user_id`: User identifier
- `resource_type`: OVN resource type
- `resource_id`: OVN resource ID
- `operation`: Operation being performed
- `duration_seconds`: Operation duration
- `error`: Error details if any

## Troubleshooting

### Prometheus Not Scraping

1. Check target status: http://localhost:9090/targets
2. Verify network connectivity
3. Check service discovery configuration
4. Review Prometheus logs

### Missing Dashboards

1. Check datasource configuration
2. Verify Grafana provisioning
3. Manually import from `grafana/dashboards/`

### No Traces Appearing

1. Verify tracing is enabled in application
2. Check Jaeger collector is receiving data
3. Review application logs for tracing errors

### High Memory Usage

1. Adjust retention policies
2. Reduce scrape frequency
3. Limit cardinality of metrics

## Maintenance

### Backup

```bash
# Backup Prometheus data
docker run --rm -v ovncp_prometheus-data:/data -v $(pwd):/backup alpine tar czf /backup/prometheus-backup.tar.gz -C /data .

# Backup Grafana data
docker run --rm -v ovncp_grafana-data:/data -v $(pwd):/backup alpine tar czf /backup/grafana-backup.tar.gz -C /data .
```

### Cleanup Old Data

```bash
# Prometheus admin API
curl -X POST http://localhost:9090/api/v1/admin/tsdb/clean_tombstones

# Delete old data
curl -X POST -g 'http://localhost:9090/api/v1/admin/tsdb/delete_series?match[]={job="ovncp-api"}&start=0&end=1640995200'
```

### Update Monitoring Stack

```bash
# Pull latest images
docker-compose -f docker-compose.monitoring.yml pull

# Restart services
docker-compose -f docker-compose.monitoring.yml up -d
```

## Integration with Kubernetes

For Kubernetes deployment, use the Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: ovncp-api
spec:
  selector:
    matchLabels:
      app: ovncp-api
  endpoints:
  - port: metrics
    interval: 30s
```

## Best Practices

1. **Metrics Cardinality**: Avoid high-cardinality labels
2. **Retention**: Set appropriate retention periods
3. **Alerting**: Start with few, actionable alerts
4. **Dashboards**: Focus on key business metrics
5. **Sampling**: Use trace sampling in production
6. **Security**: Secure monitoring endpoints

## Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)