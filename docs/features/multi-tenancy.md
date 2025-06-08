# Multi-Tenancy in OVN Control Platform

## Overview

OVN Control Platform provides comprehensive multi-tenancy support, allowing multiple organizations, projects, and teams to share the same platform while maintaining complete isolation and security.

## Key Features

### 1. Hierarchical Tenant Structure
- **Organizations**: Top-level tenants representing companies or divisions
- **Projects**: Applications or services within an organization
- **Environments**: Development, staging, production environments within projects

### 2. Resource Isolation
- Complete separation of network resources between tenants
- Automatic tenant association for all created resources
- Cross-tenant resource access prevention

### 3. Access Control
- Role-based permissions within each tenant:
  - **Admin**: Full control over tenant and resources
  - **Operator**: Create, update, and delete resources
  - **Viewer**: Read-only access
- Tenant-scoped API keys for automation

### 4. Resource Management
- Configurable quotas per tenant
- Resource usage tracking and reporting
- Automatic resource naming with tenant prefixes
- Bulk resource migration between tenants

### 5. Audit and Compliance
- Tenant-specific audit logging
- Activity tracking per tenant
- Compliance reporting capabilities

## Architecture

### Components

1. **Tenant Service** (`internal/services/tenant_service.go`)
   - Manages tenant lifecycle
   - Handles memberships and permissions
   - Tracks resource associations

2. **Tenant Middleware** (`internal/middleware/tenant.go`)
   - Extracts tenant context from requests
   - Validates tenant access permissions
   - Enforces tenant isolation

3. **Tenant-Aware OVN Service** (`internal/services/tenant_ovn_service.go`)
   - Wraps OVN operations with tenant context
   - Enforces quotas and naming conventions
   - Manages resource associations

4. **Database Layer** (`internal/db/tenant_db.go`)
   - Persistent storage for tenant data
   - Resource association tracking
   - Usage metrics storage

### Request Flow

```
Client Request
    ↓
Authentication Middleware
    ↓
Tenant Context Middleware
    ↓
Permission Check
    ↓
Tenant-Aware Service
    ↓
OVN Operations (with tenant context)
    ↓
Resource Association
    ↓
Audit Logging
```

## Implementation Details

### Tenant Context

Tenant context can be provided in three ways:

1. **HTTP Header**: `X-Tenant-ID: tenant-123`
2. **Query Parameter**: `?tenant_id=tenant-123`
3. **API Key**: Automatically associated with tenant

### Resource Naming

When `network_name_prefix` is configured:
- Input: `web-tier`
- Stored in OVN: `prod-web-tier` (with prefix)
- Returned to user: `web-tier` (without prefix)

### Quota Enforcement

Quotas are checked before resource creation:
```go
// Check quota before creating
err := tenantService.CheckQuota(ctx, tenantID, "switch", 1)
if err != nil {
    return err // Quota exceeded
}

// Create resource
switch := createSwitch(...)

// Associate with tenant
tenantService.AssociateResource(ctx, tenantID, switch.ID, "switch")
```

### External ID Management

Resources are associated with tenants using OVN external IDs:
```
external_ids: {
    "tenant_id": "tenant-123",
    "created_by": "user-456",
    "created_at": "2024-01-20T10:00:00Z"
}
```

## Usage Examples

### Creating a Tenant Hierarchy

```bash
# Create organization
ORG_ID=$(ovncp-tenant tenant create \
  --name "acme-corp" \
  --type "organization" \
  --max-switches 1000 \
  --max-routers 200 | jq -r '.id')

# Create project under organization
PROJECT_ID=$(ovncp-tenant tenant create \
  --name "web-app" \
  --type "project" \
  --parent $ORG_ID \
  --max-switches 100 | jq -r '.id')

# Create environments
ovncp-tenant tenant create \
  --name "dev" \
  --type "environment" \
  --parent $PROJECT_ID \
  --max-switches 10

ovncp-tenant tenant create \
  --name "prod" \
  --type "environment" \
  --parent $PROJECT_ID \
  --max-switches 50
```

### Managing Members

```bash
# Add team member
ovncp-tenant member add $PROJECT_ID user-123 --role operator

# List members
ovncp-tenant member list $PROJECT_ID

# Update role
curl -X PUT $OVNCP_URL/api/v1/tenants/$PROJECT_ID/members/user-123/role \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"role": "admin"}'
```

### Creating Resources in Tenant Context

```bash
# Create switch in tenant context
curl -X POST $OVNCP_URL/api/v1/switches \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "web-tier",
    "description": "Web application tier"
  }'

# List resources in tenant
curl $OVNCP_URL/api/v1/switches \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### API Key Management

```bash
# Create API key
KEY=$(ovncp-tenant apikey create $TENANT_ID \
  --name "CI/CD Pipeline" \
  --scopes read,write \
  --expires-in 90)

# Use API key
curl $OVNCP_URL/api/v1/switches \
  -H "Authorization: Bearer $KEY"
```

## Migration Guide

### Migrating Existing Resources

For existing deployments, use the migration script:

```python
# migrate_to_tenants.py
import requests
import json

def migrate_resources(api_url, token, tenant_mapping):
    """Migrate existing resources to tenants based on mapping"""
    
    # Get all switches
    switches = get_all_switches(api_url, token)
    
    for switch in switches:
        # Determine tenant based on naming convention or tags
        tenant_id = determine_tenant(switch, tenant_mapping)
        
        # Update switch with tenant external ID
        update_switch_tenant(api_url, token, switch['id'], tenant_id)
```

### Gradual Migration Steps

1. **Phase 1**: Create tenant structure
2. **Phase 2**: Update API clients to include tenant context
3. **Phase 3**: Migrate existing resources
4. **Phase 4**: Enable tenant enforcement

## Best Practices

### 1. Tenant Naming
- Use consistent, meaningful names
- Include environment in naming (e.g., `prod-web-app`)
- Avoid special characters

### 2. Quota Planning
- Set realistic quotas based on expected usage
- Leave headroom for growth
- Monitor usage regularly

### 3. Access Management
- Follow principle of least privilege
- Use role hierarchy appropriately
- Regularly audit memberships

### 4. API Key Security
- Set expiration dates
- Use minimal required scopes
- Rotate keys regularly
- Store securely (e.g., in secrets manager)

### 5. Resource Organization
- Use consistent tagging
- Group related resources
- Document resource ownership

## Performance Considerations

### Caching
- Tenant metadata is cached for 5 minutes
- Membership checks are cached per request
- Resource counts are updated asynchronously

### Database Indexes
Required indexes for optimal performance:
```sql
CREATE INDEX idx_tenant_members_tenant_user ON tenant_memberships(tenant_id, user_id);
CREATE INDEX idx_tenant_resources_tenant_type ON tenant_resources(tenant_id, resource_type);
CREATE INDEX idx_tenant_resources_resource ON tenant_resources(resource_id);
```

### Scaling
- Tenant service is stateless and horizontally scalable
- Database connection pooling is recommended
- Consider read replicas for large deployments

## Troubleshooting

### Common Issues

1. **"Tenant context required"**
   - Ensure tenant ID is provided via header or query parameter
   - Check API key has tenant association

2. **"Access denied to tenant"**
   - Verify user is member of tenant
   - Check user has appropriate role
   - Ensure tenant is active

3. **"Quota exceeded"**
   - Check current usage: `ovncp-tenant tenant usage $TENANT_ID`
   - Request quota increase or clean up resources
   - Consider resource optimization

4. **"Resource not found"**
   - Verify resource exists in tenant context
   - Check tenant isolation is working correctly
   - Ensure proper tenant ID is being used

### Debug Mode

Enable debug logging for tenant operations:
```yaml
logging:
  level: debug
  modules:
    tenant: debug
    middleware.tenant: debug
```

## Security Considerations

### Isolation Guarantees
- Resources are filtered at API level
- OVN operations include tenant validation
- Cross-tenant references are prevented

### Audit Trail
- All tenant operations are logged
- Resource creation includes tenant metadata
- API key usage is tracked

### Compliance
- Tenant data can be exported for compliance
- Resource usage reports available
- Audit logs maintained per tenant

## Future Enhancements

1. **Cost Tracking**
   - Resource usage billing
   - Cost allocation per tenant
   - Budget alerts

2. **Advanced Quotas**
   - Time-based quotas
   - Burst capacity
   - Resource reservation

3. **Federation**
   - Cross-region tenant support
   - Federated authentication
   - Global resource management

4. **Self-Service Portal**
   - Tenant management UI
   - Resource visualization
   - Usage dashboards