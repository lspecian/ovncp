# OVN Flow Tracing

The OVN Control Platform provides powerful flow tracing capabilities to help debug network connectivity issues and understand packet flow through your OVN network.

## Overview

Flow tracing allows you to:
- Trace the path of a packet through the OVN logical network
- Identify where packets are being dropped
- Understand which ACL rules are affecting traffic
- Debug connectivity issues between endpoints
- Analyze allowed protocols and ports between nodes

## Features

### Single Flow Trace

Trace a single packet flow through the network to see exactly how it would be processed.

```http
POST /api/v1/trace/flow
```

Request body:
```json
{
  "source_port": "vm1-port",
  "source_mac": "00:00:00:00:00:01",
  "source_ip": "10.0.0.1",
  "destination_mac": "00:00:00:00:00:02",
  "destination_ip": "10.0.0.2",
  "protocol": "tcp",
  "source_port_num": 12345,
  "destination_port_num": 80,
  "verbose": true
}
```

Response:
```json
{
  "request": { ... },
  "success": true,
  "reaches_destination": true,
  "hops": [
    {
      "index": 0,
      "type": "port_security",
      "component": "ingress(dp=\"ls1\", inport=\"vm1-port\")",
      "action": "forward",
      "description": "Port security check - MAC and IP validation"
    },
    {
      "index": 1,
      "type": "acl",
      "component": "ingress ACL",
      "action": "forward",
      "description": "Allow HTTP traffic",
      "acl_matches": [
        {
          "acl_name": "allow-http",
          "priority": 2000,
          "direction": "ingress",
          "action": "allow",
          "match": "tcp.dst == 80"
        }
      ]
    },
    {
      "index": 2,
      "type": "l2_lookup",
      "component": "L2 forwarding",
      "action": "output",
      "description": "L2 lookup - forward to port for MAC 00:00:00:00:00:02",
      "next_hop": "vm2-port"
    }
  ],
  "summary": "Packet successfully traced from vm1-port to 10.0.0.2 through 3 hops"
}
```

### Multi-Path Trace

Test multiple protocols and ports between two endpoints to understand what traffic is allowed.

```http
POST /api/v1/trace/multi-path
```

Request body:
```json
{
  "source_port": "vm1-port",
  "source_mac": "00:00:00:00:00:01",
  "source_ip": "10.0.0.1",
  "destination_ip": "10.0.0.2",
  "protocols": ["tcp", "udp", "icmp"],
  "ports": [22, 80, 443, 3306]
}
```

Response:
```json
{
  "source_port": "vm1-port",
  "destination_ip": "10.0.0.2",
  "paths": [
    {
      "protocol": "tcp",
      "port": 22,
      "reaches_destination": false,
      "blocked": true,
      "blocked_by": "ACL: deny-ssh (Priority: 2100)",
      "hop_count": 2
    },
    {
      "protocol": "tcp",
      "port": 80,
      "reaches_destination": true,
      "blocked": false,
      "hop_count": 3
    },
    {
      "protocol": "tcp",
      "port": 443,
      "reaches_destination": true,
      "blocked": false,
      "hop_count": 3
    },
    {
      "protocol": "icmp",
      "reaches_destination": true,
      "blocked": false,
      "hop_count": 3
    }
  ],
  "summary": "2 of 5 paths blocked. Allowed: tcp:80, tcp:443, tcp:3306, icmp"
}
```

### Connectivity Analysis

Analyze connectivity between a source port and multiple target ports.

```http
POST /api/v1/trace/connectivity
```

Request body:
```json
{
  "source_port": "web-server-port",
  "target_ports": ["db-server-port", "cache-server-port", "app-server-port"]
}
```

Response:
```json
{
  "source_port": "web-server-port",
  "target_ports": [
    {
      "port_name": "db-server-port",
      "port_ip": "10.0.2.10",
      "reachable": true,
      "protocols": {
        "icmp": true,
        "tcp:3306": true
      }
    },
    {
      "port_name": "cache-server-port",
      "port_ip": "10.0.3.10",
      "reachable": true,
      "protocols": {
        "icmp": true,
        "tcp:6379": true
      }
    },
    {
      "port_name": "app-server-port",
      "port_ip": "10.0.1.20",
      "reachable": false,
      "protocols": {}
    }
  ],
  "fully_reachable": false,
  "recommendations": [
    "The following ports are unreachable: app-server-port",
    "Check ACL rules and ensure proper routing between network segments"
  ]
}
```

### Get Port Addresses

Helper endpoint to get MAC and IP addresses for a port.

```http
GET /api/v1/trace/ports/:port/addresses
```

Response:
```json
{
  "port_id": "uuid-1234",
  "port_name": "vm1-port",
  "addresses": [
    {
      "mac": "00:00:00:00:00:01",
      "ip": "10.0.0.1"
    }
  ],
  "type": "internal"
}
```

### Flow Simulation

Test flow tracing without actual OVN execution (useful for development/testing).

```http
POST /api/v1/trace/simulate
```

Same request format as `/trace/flow`, but returns simulated results.

## Use Cases

### 1. Debugging Connectivity Issues

When a VM cannot reach another VM or service:

```bash
# Check if web server can reach database
curl -X POST https://ovncp.example.com/api/v1/trace/flow \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_port": "web-vm-port",
    "source_mac": "00:00:00:00:00:01",
    "source_ip": "10.0.1.10",
    "destination_ip": "10.0.2.10",
    "protocol": "tcp",
    "destination_port_num": 3306
  }'
```

### 2. Validating ACL Rules

Test if your ACL rules are working as expected:

```bash
# Test multiple protocols/ports
curl -X POST https://ovncp.example.com/api/v1/trace/multi-path \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_port": "client-port",
    "source_mac": "00:00:00:00:00:10",
    "source_ip": "192.168.1.10",
    "destination_ip": "192.168.2.20",
    "protocols": ["tcp", "udp", "icmp"],
    "ports": [22, 80, 443, 8080]
  }'
```

### 3. Pre-deployment Validation

Before deploying a new service, verify network connectivity:

```bash
# Analyze connectivity from new service to dependencies
curl -X POST https://ovncp.example.com/api/v1/trace/connectivity \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_port": "new-service-port",
    "target_ports": ["database-port", "cache-port", "api-gateway-port"]
  }'
```

### 4. Security Audit

Verify that sensitive services are properly isolated:

```bash
# Check if external ports can reach internal services
curl -X POST https://ovncp.example.com/api/v1/trace/flow \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_port": "dmz-port",
    "source_mac": "00:00:00:00:00:20",
    "source_ip": "172.16.0.10",
    "destination_ip": "10.0.0.5",
    "protocol": "tcp",
    "destination_port_num": 22
  }'
```

## Understanding Flow Trace Results

### Hop Types

- **port_security**: Port security validation (MAC/IP binding)
- **acl**: Access Control List evaluation
- **l2_lookup**: Layer 2 (MAC) address lookup
- **l3_routing**: Layer 3 (IP) routing decision
- **nat**: Network Address Translation
- **load_balancer**: Load balancer processing
- **flow**: Generic flow processing

### Actions

- **forward**: Packet continues to next stage
- **drop**: Packet is dropped (blocked)
- **output**: Packet is sent to destination port
- **modify**: Packet headers are modified (NAT, etc.)

### Common Drop Reasons

1. **ACL Block**: Packet matched a deny ACL rule
2. **Port Security Violation**: Source MAC/IP doesn't match port binding
3. **No Route**: No route exists to destination
4. **Invalid Destination**: Destination MAC/IP not found

## Best Practices

### 1. Use Verbose Mode

Enable verbose mode for detailed trace information:

```json
{
  "verbose": true,
  "max_hops": 50
}
```

### 2. Test Both Directions

Network flows are often asymmetric. Test both directions:

```bash
# Test A -> B
curl -X POST .../trace/flow -d '{"source_port": "A", "destination_ip": "B", ...}'

# Test B -> A  
curl -X POST .../trace/flow -d '{"source_port": "B", "destination_ip": "A", ...}'
```

### 3. Test Multiple Protocols

Different protocols may have different ACL rules:

```bash
# Use multi-path trace
curl -X POST .../trace/multi-path -d '{
  "protocols": ["tcp", "udp", "icmp"],
  "ports": [22, 80, 443, 3306, 5432]
}'
```

### 4. Automate Testing

Create scripts to regularly test critical paths:

```python
import requests
import json

critical_paths = [
    ("web-tier", "app-tier", ["tcp:8080", "tcp:8443"]),
    ("app-tier", "db-tier", ["tcp:3306", "tcp:6379"]),
    ("app-tier", "external-api", ["tcp:443"])
]

for source, dest, protocols in critical_paths:
    result = trace_connectivity(source, dest, protocols)
    if not result['fully_reachable']:
        alert_team(f"Connectivity issue: {source} -> {dest}")
```

## Troubleshooting

### "Source port not found"

Ensure the port name or UUID is correct:
```bash
# List ports to find correct name
curl -X GET https://ovncp.example.com/api/v1/switches/SWITCH_ID/ports
```

### "Invalid MAC/IP address"

Addresses must be in correct format:
- MAC: `XX:XX:XX:XX:XX:XX` (colon-separated hex)
- IPv4: `X.X.X.X` (dot notation)
- IPv6: Standard IPv6 format

### No trace output

1. Check OVN connectivity
2. Ensure ovn-trace is available on the system
3. Check service logs for errors

### Trace shows unexpected results

1. Verify ACL rules are correctly configured
2. Check for overlapping or conflicting rules
3. Ensure port security settings are correct
4. Verify routing configuration

## Performance Considerations

### Caching

Flow trace results are NOT cached as network state can change. Each trace reflects current network configuration.

### Rate Limiting

Flow tracing operations are rate-limited to prevent abuse:
- Default: 10 requests per second per user
- Burst: 50 requests

### Timeouts

Long traces may timeout. Adjust timeout if needed:
- Default timeout: 30 seconds
- Maximum hops: 100

## Integration Examples

### CLI Tool

```bash
#!/bin/bash
# ovncp-trace - CLI tool for flow tracing

function trace_flow() {
    local source_port=$1
    local dest_ip=$2
    local protocol=$3
    local port=$4
    
    curl -s -X POST https://ovncp.example.com/api/v1/trace/flow \
        -H "Authorization: Bearer $OVNCP_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"source_port\": \"$source_port\",
            \"destination_ip\": \"$dest_ip\",
            \"protocol\": \"$protocol\",
            \"destination_port_num\": $port
        }" | jq .
}

# Usage: ovncp-trace web-port 10.0.2.10 tcp 3306
trace_flow "$@"
```

### Python SDK

```python
from ovncp import Client

client = Client(base_url='https://ovncp.example.com', token='...')

# Single flow trace
result = client.trace.flow(
    source_port='web-port',
    source_mac='00:00:00:00:00:01',
    source_ip='10.0.1.10',
    destination_ip='10.0.2.20',
    protocol='tcp',
    destination_port=443
)

if result.reaches_destination:
    print(f"✓ Traffic allowed through {len(result.hops)} hops")
else:
    print(f"✗ Traffic blocked at: {result.dropped_at.description}")

# Multi-path analysis
paths = client.trace.multi_path(
    source_port='web-port',
    destination_ip='10.0.2.20',
    protocols=['tcp', 'udp'],
    ports=[80, 443, 3306]
)

for path in paths.paths:
    status = "✓" if path.reaches_destination else "✗"
    print(f"{status} {path.protocol}:{path.port}")
```

### Monitoring Integration

```javascript
// Prometheus metrics exporter
async function checkCriticalPaths() {
    const paths = [
        { source: 'web-port', dest: '10.0.2.10', protocol: 'tcp', port: 3306 },
        { source: 'app-port', dest: '10.0.3.10', protocol: 'tcp', port: 6379 }
    ];
    
    for (const path of paths) {
        const result = await traceFlow(path);
        
        // Export metric
        ovncp_path_reachable.labels({
            source: path.source,
            destination: path.dest,
            protocol: path.protocol,
            port: path.port
        }).set(result.reaches_destination ? 1 : 0);
    }
}

// Run every 5 minutes
setInterval(checkCriticalPaths, 300000);
```