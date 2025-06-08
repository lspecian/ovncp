# Network Policy Templates

The OVN Control Platform provides a powerful policy template system that allows you to create reusable network security policies. Templates help standardize security configurations across your infrastructure and reduce configuration errors.

## Overview

Policy templates allow you to:
- Define reusable security policies with customizable variables
- Apply consistent security patterns across multiple switches
- Validate configurations before applying them
- Import/export templates for sharing and backup
- Generate complex ACL rules from simple parameters

## Built-in Templates

### 1. Web Server Template

Standard security policy for web servers supporting HTTP/HTTPS traffic.

**Variables:**
- `server_ip` (required): IP address of the web server
- `allowed_sources`: CIDR blocks allowed to access (default: 0.0.0.0/0)
- `enable_ssh`: Allow SSH access for management (default: false)
- `ssh_sources`: CIDR blocks allowed for SSH (default: 10.0.0.0/8)

**Example:**
```json
{
  "template_id": "web-server",
  "variables": {
    "server_ip": "10.0.1.10",
    "allowed_sources": "192.168.0.0/16",
    "enable_ssh": true,
    "ssh_sources": "10.0.100.0/24"
  }
}
```

### 2. Database Server Template

Security policy for database servers (MySQL/PostgreSQL).

**Variables:**
- `db_ip` (required): IP address of the database server
- `db_port` (required): Database port (default: 3306)
- `app_subnet` (required): Application server subnet
- `backup_server`: Backup server IP address
- `enable_replication`: Enable database replication
- `replica_ips`: Comma-separated list of replica IPs

**Example:**
```json
{
  "template_id": "database-server",
  "variables": {
    "db_ip": "10.0.2.10",
    "db_port": 5432,
    "app_subnet": "10.0.1.0/24",
    "backup_server": "10.0.100.50",
    "enable_replication": true,
    "replica_ips": "10.0.2.11,10.0.2.12"
  }
}
```

### 3. Microservice Template

Security policy for microservices with service mesh support.

**Variables:**
- `service_name` (required): Name of the microservice
- `service_ip` (required): IP address of the service
- `service_port` (required): Service port (default: 8080)
- `health_port`: Health check port (default: 8081)
- `allowed_services` (required): Comma-separated list of allowed source IPs
- `metrics_port`: Metrics/Prometheus port (default: 9090)
- `monitoring_subnet`: Monitoring system subnet

**Example:**
```json
{
  "template_id": "microservice",
  "variables": {
    "service_name": "user-service",
    "service_ip": "10.0.3.10",
    "service_port": 8080,
    "allowed_services": "10.0.3.11,10.0.3.12,10.0.3.13",
    "monitoring_subnet": "10.0.200.0/24"
  }
}
```

### 4. DMZ Zone Template

Security policy for DMZ (Demilitarized Zone) networks.

**Variables:**
- `dmz_subnet` (required): DMZ subnet CIDR
- `internal_subnets` (required): Internal network subnets (comma-separated)
- `allowed_dmz_to_internal_ports`: Ports DMZ can access internally

**Example:**
```json
{
  "template_id": "dmz",
  "variables": {
    "dmz_subnet": "172.16.0.0/24",
    "internal_subnets": "10.0.0.0/16,192.168.0.0/16",
    "allowed_dmz_to_internal_ports": "443,636,389"
  }
}
```

### 5. Zero Trust Template

Zero trust security model - deny by default, explicit allow.

**Variables:**
- `resource_ip` (required): IP of the protected resource
- `resource_port` (required): Port of the protected resource
- `authorized_users` (required): Authorized user IPs (comma-separated)
- `require_encryption`: Require encrypted connections only (default: true)

**Example:**
```json
{
  "template_id": "zero-trust",
  "variables": {
    "resource_ip": "10.0.5.10",
    "resource_port": 443,
    "authorized_users": "10.0.100.10,10.0.100.11",
    "require_encryption": true
  }
}
```

### 6. Kubernetes Pod Template

Network policy for Kubernetes pods.

**Variables:**
- `pod_cidr` (required): Pod network CIDR
- `service_cidr` (required): Service network CIDR
- `namespace` (required): Kubernetes namespace
- `pod_selector` (required): Pod label selector

**Example:**
```json
{
  "template_id": "k8s-pod",
  "variables": {
    "pod_cidr": "10.244.0.0/16",
    "service_cidr": "10.96.0.0/12",
    "namespace": "production",
    "pod_selector": "app=nginx"
  }
}
```

## API Endpoints

### List Templates

```http
GET /api/v1/templates
```

Query parameters:
- `category`: Filter by category (e.g., "Application", "Network Zone")
- `tag[]`: Filter by tags (can specify multiple)

Response:
```json
{
  "templates": [
    {
      "id": "web-server",
      "name": "Web Server",
      "description": "Standard security policy for web servers",
      "category": "Application",
      "tags": ["web", "http", "https"],
      "variables": [...]
    }
  ]
}
```

### Get Template Details

```http
GET /api/v1/templates/:id
```

Response includes full template definition with variables and rules.

### Validate Template

Test template variables before applying:

```http
POST /api/v1/templates/validate
```

Request:
```json
{
  "template_id": "web-server",
  "variables": {
    "server_ip": "10.0.1.10",
    "allowed_sources": "192.168.0.0/16"
  }
}
```

Response:
```json
{
  "valid": true,
  "errors": {},
  "warnings": [],
  "preview": [
    {
      "name": "allow-http",
      "direction": "ingress",
      "priority": 2000,
      "match": "ip4.dst == 10.0.1.10 && tcp.dst == 80 && ip4.src == 192.168.0.0/16",
      "action": "allow"
    }
  ]
}
```

### Instantiate Template

Create ACL rules from a template:

```http
POST /api/v1/templates/instantiate
```

Request:
```json
{
  "template_id": "web-server",
  "variables": {
    "server_ip": "10.0.1.10",
    "allowed_sources": "192.168.0.0/16",
    "enable_ssh": true
  },
  "target_switch": "ls-web-tier",
  "dry_run": false
}
```

Response:
```json
{
  "instance": {
    "template_id": "web-server",
    "name": "Web Server",
    "variables": {...},
    "rules": [
      {
        "name": "allow-http",
        "direction": "ingress",
        "priority": 2000,
        "match": "ip4.dst == 10.0.1.10 && tcp.dst == 80 && ip4.src == 192.168.0.0/16",
        "action": "allow"
      }
    ]
  },
  "message": "Template instantiated successfully"
}
```

### Import Custom Template

```http
POST /api/v1/templates/import
```

Request body should contain a valid template JSON.

### Export Template

```http
GET /api/v1/templates/:id/export
```

Downloads the template as a JSON file.

## Creating Custom Templates

### Template Structure

```json
{
  "id": "custom-app",
  "name": "Custom Application",
  "description": "Security policy for custom application",
  "category": "Application",
  "tags": ["custom", "app"],
  "variables": [
    {
      "name": "app_ip",
      "description": "Application IP address",
      "type": "ipv4",
      "required": true,
      "example": "10.0.1.10"
    },
    {
      "name": "app_port",
      "description": "Application port",
      "type": "port",
      "required": false,
      "default": 8080
    }
  ],
  "rules": [
    {
      "name": "allow-app-traffic",
      "description": "Allow application traffic",
      "priority": 2000,
      "direction": "ingress",
      "action": "allow",
      "match": "ip4.dst == {{app_ip}} && tcp.dst == {{app_port}}",
      "log": false
    }
  ]
}
```

### Variable Types

- `string`: Text value
- `number`: Numeric value
- `boolean`: True/false value
- `ipv4`: IPv4 address
- `ipv6`: IPv6 address
- `cidr`: CIDR notation
- `port`: Port number (1-65535)
- `mac`: MAC address

### Template Syntax

Templates use Go template syntax with custom functions:

**Basic variable substitution:**
```
{{variable_name}}
```

**Conditional logic:**
```
{{if enable_ssh}}...{{else}}0{{end}}
```

**List formatting (triple braces):**
```
{{{comma_separated_ips}}}
```
Converts "10.0.1.1,10.0.1.2" to OVN format: {10.0.1.1, 10.0.1.2}

**Available functions:**
- `join`: Join array elements
- `split`: Split string
- `contains`: Check if string contains substring
- `hasPrefix`: Check string prefix
- `hasSuffix`: Check string suffix

## Best Practices

### 1. Variable Naming

Use descriptive, consistent variable names:
- `server_ip` not `ip`
- `allowed_sources` not `src`
- `enable_feature` not `feature`

### 2. Default Values

Provide sensible defaults for optional variables:
```json
{
  "name": "monitoring_port",
  "type": "port",
  "required": false,
  "default": 9090
}
```

### 3. Documentation

Always include:
- Clear descriptions for variables
- Examples for complex variables
- Rule descriptions explaining purpose

### 4. Security Considerations

- Always include a default deny rule at low priority
- Use specific match criteria, avoid overly broad rules
- Enable logging for security-relevant rules
- Test templates thoroughly before production use

### 5. Template Categories

Organize templates by category:
- **Application**: Service-specific policies
- **Network Zone**: Zone-based policies (DMZ, internal, etc.)
- **Security Model**: Overall security approaches
- **Container**: Container/orchestration policies

## Troubleshooting

### Common Issues

**"Required variable not provided"**
- Ensure all required variables are included in the request
- Check variable names match exactly (case-sensitive)

**"Invalid IPv4 address"**
- Verify IP addresses are in correct format (X.X.X.X)
- Check for typos or extra spaces

**"Template validation failed"**
- Review validation errors in response
- Ensure variable types match expected types
- Check CIDR notation is correct (X.X.X.X/Y)

**"Template not found"**
- Verify template ID is correct
- Check if template has been imported (for custom templates)

### Debugging Templates

1. **Use dry_run mode:**
   ```json
   {
     "dry_run": true
   }
   ```
   Preview rules without creating them.

2. **Validate first:**
   Always validate templates before instantiation.

3. **Check generated rules:**
   Review the preview to ensure rules match expectations.

4. **Test incrementally:**
   Start with minimal variables, add complexity gradually.

## Examples

### Secure Web Application

Deploy a web application with database backend:

```bash
# 1. Create web server policy
curl -X POST $OVNCP_URL/api/v1/templates/instantiate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "template_id": "web-server",
    "variables": {
      "server_ip": "10.0.1.10",
      "allowed_sources": "0.0.0.0/0",
      "enable_ssh": true,
      "ssh_sources": "10.0.100.0/24"
    },
    "target_switch": "ls-web"
  }'

# 2. Create database policy
curl -X POST $OVNCP_URL/api/v1/templates/instantiate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "template_id": "database-server",
    "variables": {
      "db_ip": "10.0.2.10",
      "db_port": 3306,
      "app_subnet": "10.0.1.0/24"
    },
    "target_switch": "ls-db"
  }'
```

### Microservices Mesh

Configure policies for interconnected microservices:

```python
import requests

services = [
    {"name": "user-service", "ip": "10.0.3.10", "deps": ["10.0.3.11", "10.0.3.12"]},
    {"name": "order-service", "ip": "10.0.3.11", "deps": ["10.0.3.10", "10.0.3.12"]},
    {"name": "payment-service", "ip": "10.0.3.12", "deps": ["10.0.3.11"]}
]

for service in services:
    response = requests.post(
        f"{OVNCP_URL}/api/v1/templates/instantiate",
        headers={"Authorization": f"Bearer {TOKEN}"},
        json={
            "template_id": "microservice",
            "variables": {
                "service_name": service["name"],
                "service_ip": service["ip"],
                "allowed_services": ",".join(service["deps"]),
                "monitoring_subnet": "10.0.200.0/24"
            },
            "target_switch": "ls-microservices"
        }
    )
```

### Zero Trust Implementation

Implement zero trust for sensitive resources:

```javascript
const resources = [
  { ip: "10.0.5.10", port: 443, users: ["10.0.100.10", "10.0.100.11"] },
  { ip: "10.0.5.11", port: 22, users: ["10.0.100.10"] }
];

for (const resource of resources) {
  await fetch(`${OVNCP_URL}/api/v1/templates/instantiate`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      template_id: 'zero-trust',
      variables: {
        resource_ip: resource.ip,
        resource_port: resource.port,
        authorized_users: resource.users.join(','),
        require_encryption: true
      },
      target_switch: 'ls-secure'
    })
  });
}