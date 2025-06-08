# Multi-Tenancy

The OVN Control Platform provides comprehensive multi-tenancy support, allowing multiple organizations or projects to share the same platform while maintaining complete isolation and security.

## Overview

Multi-tenancy in OVNCP provides:
- **Resource Isolation**: Complete separation of network resources between tenants
- **Access Control**: Role-based permissions within each tenant
- **Resource Quotas**: Configurable limits per tenant
- **Audit Trail**: Tenant-specific audit logging
- **API Keys**: Tenant-scoped API access
- **Hierarchical Organization**: Support for organizations, projects, and environments

## Tenant Types

### 1. Organization
Top-level tenant representing a company or organization.

### 2. Project
A project within an organization, typically representing an application or team.

### 3. Environment
An environment within a project (e.g., development, staging, production).

## Getting Started

### Creating a Tenant

```bash
curl -X POST $OVNCP_URL/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-company",
    "display_name": "My Company Inc.",
    "description": "Main organization tenant",
    "type": "organization",
    "settings": {
      "network_name_prefix": "myco",
      "enable_audit_logging": true
    }
  }'
```

### Listing Your Tenants

```bash
curl $OVNCP_URL/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN"
```

### Setting Tenant Context

All API operations can be scoped to a specific tenant:

```bash
# Using header
curl $OVNCP_URL/api/v1/switches \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"

# Using query parameter
curl $OVNCP_URL/api/v1/switches?tenant_id=tenant-123 \
  -H "Authorization: Bearer $TOKEN"
```

## Tenant Management

### Tenant Settings

Configure tenant-specific behaviors:

```json
{
  "settings": {
    "default_network_type": "overlay",
    "network_name_prefix": "prod",
    "require_approval": true,
    "allow_external_networks": false,
    "enable_audit_logging": true,
    "custom_labels": {
      "environment": "production",
      "cost_center": "engineering"
    }
  }
}
```

### Resource Quotas

Set limits on resource creation:

```json
{
  "quotas": {
    "max_switches": 100,
    "max_routers": 50,
    "max_ports": 1000,
    "max_acls": 500,
    "max_load_balancers": 20,
    "max_address_sets": 100,
    "max_port_groups": 100,
    "max_backups": 50
  }
}
```

Use `-1` for unlimited resources.

### Updating a Tenant

```bash
curl -X PUT $OVNCP_URL/api/v1/tenants/$TENANT_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "My Company International",
    "quotas": {
      "max_switches": 200,
      "max_routers": 100
    }
  }'
```

## Member Management

### Roles

Each tenant member has one of these roles:

- **admin**: Full control over tenant and its resources
- **operator**: Create, update, and delete resources
- **viewer**: Read-only access to resources

### Adding Members

```bash
curl -X POST $OVNCP_URL/api/v1/tenants/$TENANT_ID/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-456",
    "role": "operator"
  }'
```

### Listing Members

```bash
curl $OVNCP_URL/api/v1/tenants/$TENANT_ID/members \
  -H "Authorization: Bearer $TOKEN"
```

### Updating Member Role

```bash
curl -X PUT $OVNCP_URL/api/v1/tenants/$TENANT_ID/members/$USER_ID/role \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "admin"
  }'
```

### Removing Members

```bash
curl -X DELETE $OVNCP_URL/api/v1/tenants/$TENANT_ID/members/$USER_ID \
  -H "Authorization: Bearer $TOKEN"
```

## Invitations

Invite users to join your tenant via email:

### Creating an Invitation

```bash
curl -X POST $OVNCP_URL/api/v1/tenants/$TENANT_ID/invitations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "role": "operator"
  }'
```

### Accepting an Invitation

```bash
curl -X POST $OVNCP_URL/api/v1/invitations/$TOKEN/accept \
  -H "Authorization: Bearer $USER_TOKEN"
```

## API Keys

Create tenant-scoped API keys for automation:

### Creating an API Key

```bash
curl -X POST $OVNCP_URL/api/v1/tenants/$TENANT_ID/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD Pipeline",
    "description": "API key for automated deployments",
    "scopes": ["read", "write"],
    "expires_in": 90
  }'
```

Response:
```json
{
  "key": "ovncp_12345678_abcdefghijklmnopqrstuvwxyz123456",
  "message": "API key created successfully. Please save the key, it won't be shown again."
}
```

### Using API Keys

```bash
# In Authorization header
curl $OVNCP_URL/api/v1/switches \
  -H "Authorization: Bearer ovncp_12345678_abcdefghijklmnopqrstuvwxyz123456"

# In X-API-Key header
curl $OVNCP_URL/api/v1/switches \
  -H "X-API-Key: ovncp_12345678_abcdefghijklmnopqrstuvwxyz123456"
```

### Listing API Keys

```bash
curl $OVNCP_URL/api/v1/tenants/$TENANT_ID/api-keys \
  -H "Authorization: Bearer $TOKEN"
```

### Deleting API Keys

```bash
curl -X DELETE $OVNCP_URL/api/v1/tenants/$TENANT_ID/api-keys/$KEY_ID \
  -H "Authorization: Bearer $TOKEN"
```

## Resource Management

### Automatic Tenant Association

When creating resources within a tenant context, they are automatically associated:

```bash
# This switch will belong to tenant-123
curl -X POST $OVNCP_URL/api/v1/switches \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-tier",
    "description": "Web application tier"
  }'
```

### Resource Naming

If configured, tenant prefix is automatically added:

```json
{
  "settings": {
    "network_name_prefix": "prod"
  }
}
```

Creating a switch named "web-tier" will result in "prod-web-tier".

### Resource Filtering

Resources are automatically filtered by tenant:

```bash
# Only shows resources belonging to the tenant
curl $OVNCP_URL/api/v1/switches \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"
```

### Cross-Tenant Access

Resources from one tenant cannot be accessed from another tenant context, even if the user has access to both tenants.

## Resource Usage and Quotas

### Checking Usage

```bash
curl $OVNCP_URL/api/v1/tenants/$TENANT_ID/usage \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "usage": {
    "switches": 45,
    "routers": 12,
    "ports": 324,
    "acls": 156,
    "load_balancers": 8,
    "address_sets": 23,
    "port_groups": 15,
    "backups": 7
  },
  "quotas": {
    "max_switches": 100,
    "max_routers": 50,
    "max_ports": 1000,
    "max_acls": 500,
    "max_load_balancers": 20,
    "max_address_sets": 100,
    "max_port_groups": 100,
    "max_backups": 50
  }
}
```

### Quota Enforcement

When a quota is reached:

```json
{
  "error": "quota exceeded: switches (current: 100, limit: 100)"
}
```

## Hierarchical Tenants

Create a hierarchy of tenants:

```bash
# Create organization
ORG_ID=$(curl -X POST $OVNCP_URL/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "acme-corp",
    "type": "organization"
  }' | jq -r '.id')

# Create project under organization
PROJECT_ID=$(curl -X POST $OVNCP_URL/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "web-app",
    "type": "project",
    "parent": "'$ORG_ID'"
  }' | jq -r '.id')

# Create environment under project
ENV_ID=$(curl -X POST $OVNCP_URL/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "production",
    "type": "environment",
    "parent": "'$PROJECT_ID'"
  }' | jq -r '.id')
```

## Best Practices

### 1. Tenant Naming

Use clear, consistent naming:
- Organizations: Company name (e.g., "acme-corp")
- Projects: Application or team name (e.g., "customer-portal")
- Environments: Standard names (e.g., "dev", "staging", "prod")

### 2. Resource Prefixes

Configure prefixes to avoid naming conflicts:

```json
{
  "settings": {
    "network_name_prefix": "acme-prod"
  }
}
```

### 3. Quota Planning

Set quotas based on expected usage:
- Development: Lower quotas
- Production: Higher quotas with headroom
- Shared environments: Strict quotas

### 4. Access Control

Follow principle of least privilege:
- Developers: `operator` role in dev, `viewer` in prod
- DevOps: `operator` role in all environments
- Admins: `admin` role at organization level

### 5. API Key Management

- Use descriptive names
- Set expiration dates
- Rotate regularly
- Use minimal required scopes
- Store securely

## Migration Guide

### Migrating Existing Resources

To migrate existing resources to multi-tenant:

1. Create tenant structure
2. Associate existing resources
3. Update API calls to include tenant context

```python
import requests

# Migrate switches to tenant
def migrate_to_tenant(api_url, token, tenant_id):
    # Get all switches
    response = requests.get(
        f"{api_url}/api/v1/switches",
        headers={"Authorization": f"Bearer {token}"}
    )
    switches = response.json()
    
    # Associate each switch with tenant
    for switch in switches:
        # Update switch with tenant external ID
        requests.put(
            f"{api_url}/api/v1/switches/{switch['id']}",
            headers={
                "Authorization": f"Bearer {token}",
                "X-Tenant-ID": tenant_id
            },
            json={
                "external_ids": {
                    **switch.get('external_ids', {}),
                    "tenant_id": tenant_id
                }
            }
        )
```

### Gradual Migration

1. **Phase 1**: Create tenant structure
2. **Phase 2**: Start using tenant context for new resources
3. **Phase 3**: Migrate existing resources
4. **Phase 4**: Enforce tenant context requirement

## Security Considerations

### Isolation

- Resources are completely isolated between tenants
- No cross-tenant network connectivity by default
- Separate audit logs per tenant

### API Key Security

- Keys are hashed before storage
- Include tenant ID in key format
- Support expiration and rotation
- Track last usage

### Audit Trail

All tenant operations are logged:
- Member changes
- Resource creation/deletion
- Configuration updates
- API key usage

## Troubleshooting

### "Tenant context required"

Ensure tenant ID is provided via:
- `X-Tenant-ID` header
- `tenant_id` query parameter
- API key with tenant association

### "Access denied to tenant"

Verify:
- User is a member of the tenant
- User has appropriate role
- Tenant is active

### "Quota exceeded"

Options:
- Delete unused resources
- Request quota increase
- Use resource more efficiently

### API Key Issues

- Verify key format: `ovncp_<prefix>_<key>`
- Check expiration date
- Ensure key hasn't been deleted
- Verify tenant association

## Examples

### Complete Tenant Setup

```bash
#!/bin/bash

# Create organization
ORG=$(curl -s -X POST $OVNCP_URL/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "startup-inc",
    "display_name": "Startup Inc.",
    "type": "organization",
    "quotas": {
      "max_switches": 500,
      "max_routers": 100
    }
  }')

ORG_ID=$(echo $ORG | jq -r '.id')

# Add team members
for email in alice@startup.com bob@startup.com; do
  curl -X POST $OVNCP_URL/api/v1/tenants/$ORG_ID/invitations \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"email\": \"$email\", \"role\": \"operator\"}"
done

# Create projects
for project in web-app mobile-app analytics; do
  curl -X POST $OVNCP_URL/api/v1/tenants \
    -H "Authorization: Bearer $TOKEN" \
    -d "{
      \"name\": \"$project\",
      \"type\": \"project\",
      \"parent\": \"$ORG_ID\"
    }"
done

# Create API key for CI/CD
API_KEY=$(curl -s -X POST $OVNCP_URL/api/v1/tenants/$ORG_ID/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "GitLab CI",
    "scopes": ["read", "write"],
    "expires_in": 365
  }' | jq -r '.key')

echo "Organization setup complete!"
echo "API Key: $API_KEY"
```

### Multi-Environment Deployment

```python
import os
import requests

class TenantManager:
    def __init__(self, api_url, token):
        self.api_url = api_url
        self.headers = {"Authorization": f"Bearer {token}"}
    
    def setup_environments(self, org_id, project_name):
        """Setup dev, staging, and prod environments"""
        
        # Create project
        project = self.create_tenant(
            name=project_name,
            tenant_type="project",
            parent=org_id
        )
        
        environments = {}
        env_configs = {
            "dev": {"max_switches": 10, "max_routers": 5},
            "staging": {"max_switches": 20, "max_routers": 10},
            "prod": {"max_switches": 50, "max_routers": 20}
        }
        
        for env_name, quotas in env_configs.items():
            env = self.create_tenant(
                name=f"{project_name}-{env_name}",
                tenant_type="environment",
                parent=project['id'],
                settings={
                    "network_name_prefix": f"{project_name}-{env_name}",
                    "enable_audit_logging": env_name == "prod"
                },
                quotas=quotas
            )
            environments[env_name] = env
            
            # Create API key for each environment
            api_key = self.create_api_key(
                env['id'],
                f"{env_name}-deployer",
                ["read", "write"] if env_name != "prod" else ["read"]
            )
            
            print(f"Environment {env_name} created with API key: {api_key}")
        
        return environments
    
    def create_tenant(self, **kwargs):
        response = requests.post(
            f"{self.api_url}/api/v1/tenants",
            headers=self.headers,
            json=kwargs
        )
        return response.json()
    
    def create_api_key(self, tenant_id, name, scopes):
        response = requests.post(
            f"{self.api_url}/api/v1/tenants/{tenant_id}/api-keys",
            headers=self.headers,
            json={"name": name, "scopes": scopes}
        )
        return response.json()['key']

# Usage
manager = TenantManager("https://ovncp.example.com", os.getenv("OVNCP_TOKEN"))
manager.setup_environments("org-123", "customer-portal")
```