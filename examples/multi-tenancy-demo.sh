#!/bin/bash

# Multi-Tenancy Demo Script
# This script demonstrates the multi-tenancy features of OVN Control Platform

BASE_URL="${OVNCP_URL:-http://localhost:8080}"
ADMIN_TOKEN="${OVNCP_TOKEN:-your-admin-token}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo -e "\n${YELLOW}=== $1 ===${NC}\n"
}

# Function to make API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    local tenant_id=$4
    
    local headers="-H 'Authorization: Bearer $ADMIN_TOKEN' -H 'Content-Type: application/json'"
    
    if [ ! -z "$tenant_id" ]; then
        headers="$headers -H 'X-Tenant-ID: $tenant_id'"
    fi
    
    if [ -z "$data" ]; then
        eval "curl -s -X $method $BASE_URL$endpoint $headers"
    else
        eval "curl -s -X $method $BASE_URL$endpoint $headers -d '$data'"
    fi
}

# Main demo flow
log_section "Multi-Tenancy Demo for OVN Control Platform"

# 1. Create organization
log_section "Step 1: Creating Organization"
log_info "Creating 'Tech Corp' organization..."

ORG_RESPONSE=$(api_call POST /api/v1/tenants '{
    "name": "tech-corp",
    "display_name": "Tech Corp International",
    "description": "Main organization for Tech Corp",
    "type": "organization",
    "settings": {
        "network_name_prefix": "techcorp",
        "enable_audit_logging": true
    },
    "quotas": {
        "max_switches": 500,
        "max_routers": 100,
        "max_ports": 2000
    }
}')

ORG_ID=$(echo $ORG_RESPONSE | jq -r '.id')
log_success "Organization created with ID: $ORG_ID"

# 2. Create projects under organization
log_section "Step 2: Creating Projects"

# Web App Project
log_info "Creating 'Web App' project..."
WEBAPP_RESPONSE=$(api_call POST /api/v1/tenants "{
    \"name\": \"web-app\",
    \"display_name\": \"Web Application\",
    \"type\": \"project\",
    \"parent\": \"$ORG_ID\",
    \"quotas\": {
        \"max_switches\": 100,
        \"max_routers\": 20
    }
}")
WEBAPP_ID=$(echo $WEBAPP_RESPONSE | jq -r '.id')
log_success "Web App project created with ID: $WEBAPP_ID"

# Mobile App Project
log_info "Creating 'Mobile App' project..."
MOBILE_RESPONSE=$(api_call POST /api/v1/tenants "{
    \"name\": \"mobile-app\",
    \"display_name\": \"Mobile Application\",
    \"type\": \"project\",
    \"parent\": \"$ORG_ID\",
    \"quotas\": {
        \"max_switches\": 50,
        \"max_routers\": 10
    }
}")
MOBILE_ID=$(echo $MOBILE_RESPONSE | jq -r '.id')
log_success "Mobile App project created with ID: $MOBILE_ID"

# 3. Create environments
log_section "Step 3: Creating Environments"

# Development environment for Web App
log_info "Creating development environment for Web App..."
DEV_RESPONSE=$(api_call POST /api/v1/tenants "{
    \"name\": \"dev\",
    \"display_name\": \"Development\",
    \"type\": \"environment\",
    \"parent\": \"$WEBAPP_ID\",
    \"settings\": {
        \"network_name_prefix\": \"webapp-dev\"
    },
    \"quotas\": {
        \"max_switches\": 10,
        \"max_routers\": 5
    }
}")
DEV_ID=$(echo $DEV_RESPONSE | jq -r '.id')
log_success "Development environment created with ID: $DEV_ID"

# Production environment for Web App
log_info "Creating production environment for Web App..."
PROD_RESPONSE=$(api_call POST /api/v1/tenants "{
    \"name\": \"prod\",
    \"display_name\": \"Production\",
    \"type\": \"environment\",
    \"parent\": \"$WEBAPP_ID\",
    \"settings\": {
        \"network_name_prefix\": \"webapp-prod\",
        \"enable_audit_logging\": true,
        \"require_approval\": true
    },
    \"quotas\": {
        \"max_switches\": 50,
        \"max_routers\": 10
    }
}")
PROD_ID=$(echo $PROD_RESPONSE | jq -r '.id')
log_success "Production environment created with ID: $PROD_ID"

# 4. Add team members
log_section "Step 4: Adding Team Members"

# Add developer to Web App project
log_info "Adding developer to Web App project..."
api_call POST /api/v1/tenants/$WEBAPP_ID/members '{
    "user_id": "dev-user-123",
    "role": "operator"
}' > /dev/null
log_success "Developer added with operator role"

# Add DevOps to organization
log_info "Adding DevOps engineer to organization..."
api_call POST /api/v1/tenants/$ORG_ID/members '{
    "user_id": "devops-user-456",
    "role": "operator"
}' > /dev/null
log_success "DevOps engineer added with operator role"

# 5. Create API keys
log_section "Step 5: Creating API Keys"

# Create API key for CI/CD in dev environment
log_info "Creating API key for CI/CD pipeline..."
APIKEY_RESPONSE=$(api_call POST /api/v1/tenants/$DEV_ID/api-keys '{
    "name": "GitLab CI/CD",
    "description": "API key for automated deployments",
    "scopes": ["read", "write"],
    "expires_in": 90
}')
API_KEY=$(echo $APIKEY_RESPONSE | jq -r '.key')
log_success "API key created: ${API_KEY:0:20}..."

# 6. Create resources in tenant context
log_section "Step 6: Creating Resources in Tenant Context"

# Create switch in development environment
log_info "Creating switch in development environment..."
SWITCH_RESPONSE=$(api_call POST /api/v1/switches '{
    "name": "web-tier",
    "description": "Web application tier switch"
}' "" $DEV_ID)
SWITCH_ID=$(echo $SWITCH_RESPONSE | jq -r '.id')
log_success "Switch created with ID: $SWITCH_ID"

# Create router in development environment
log_info "Creating router in development environment..."
ROUTER_RESPONSE=$(api_call POST /api/v1/routers '{
    "name": "app-router",
    "description": "Application router"
}' "" $DEV_ID)
ROUTER_ID=$(echo $ROUTER_RESPONSE | jq -r '.id')
log_success "Router created with ID: $ROUTER_ID"

# 7. Check resource usage
log_section "Step 7: Checking Resource Usage"

log_info "Getting resource usage for development environment..."
USAGE_RESPONSE=$(api_call GET /api/v1/tenants/$DEV_ID/usage)
echo "Resource Usage:"
echo $USAGE_RESPONSE | jq '.'

# 8. List resources with tenant filter
log_section "Step 8: Listing Resources with Tenant Context"

log_info "Listing switches in development environment..."
SWITCHES=$(api_call GET /api/v1/switches "" "" $DEV_ID)
echo "Switches in development:"
echo $SWITCHES | jq '.switches[] | {id: .id, name: .name}'

# 9. Create invitation
log_section "Step 9: Creating Team Invitation"

log_info "Creating invitation for new team member..."
INVITE_RESPONSE=$(api_call POST /api/v1/tenants/$WEBAPP_ID/invitations '{
    "email": "newdev@techcorp.com",
    "role": "viewer"
}')
log_success "Invitation sent to newdev@techcorp.com"

# 10. List tenant hierarchy
log_section "Step 10: Viewing Tenant Hierarchy"

log_info "Listing all tenants..."
TENANTS=$(api_call GET /api/v1/tenants)
echo "Tenant Hierarchy:"
echo $TENANTS | jq '.tenants[] | {id: .id, name: .name, type: .type, parent: .parent}'

# Summary
log_section "Demo Summary"
echo "Organization: Tech Corp ($ORG_ID)"
echo "├── Web App Project ($WEBAPP_ID)"
echo "│   ├── Development Environment ($DEV_ID)"
echo "│   └── Production Environment ($PROD_ID)"
echo "└── Mobile App Project ($MOBILE_ID)"
echo ""
echo "Resources created:"
echo "- 1 Switch in development environment"
echo "- 1 Router in development environment"
echo "- 2 Team members added"
echo "- 1 API key generated"
echo "- 1 Invitation sent"

log_success "Multi-tenancy demo completed successfully!"