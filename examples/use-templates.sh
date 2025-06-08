#!/bin/bash

# Example script demonstrating how to use OVN Control Platform policy templates

OVNCP_URL="${OVNCP_URL:-http://localhost:8080}"
TOKEN="${OVNCP_TOKEN:-your-token-here}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}OVN Control Platform - Policy Templates Demo${NC}"
echo "============================================"

# Function to make API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        curl -s -X "$method" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            "$OVNCP_URL/api/v1$endpoint"
    else
        curl -s -X "$method" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$OVNCP_URL/api/v1$endpoint"
    fi
}

# 1. List available templates
echo -e "\n${GREEN}1. Listing available templates:${NC}"
api_call GET "/templates" | jq -r '.templates[] | "\(.id): \(.name) - \(.description)"'

# 2. Get details of a specific template
echo -e "\n${GREEN}2. Getting web-server template details:${NC}"
api_call GET "/templates/web-server" | jq '{id, name, variables: .variables[].name}'

# 3. Validate template with variables
echo -e "\n${GREEN}3. Validating web-server template:${NC}"
VALIDATE_DATA='{
  "template_id": "web-server",
  "variables": {
    "server_ip": "10.0.1.10",
    "allowed_sources": "192.168.0.0/16",
    "enable_ssh": true,
    "ssh_sources": "10.0.100.0/24"
  }
}'

VALIDATION_RESULT=$(api_call POST "/templates/validate" "$VALIDATE_DATA")
echo "$VALIDATION_RESULT" | jq '{valid, errors, preview: .preview | length}'

# 4. Dry run template instantiation
echo -e "\n${GREEN}4. Dry run - preview rules without creating:${NC}"
DRY_RUN_DATA='{
  "template_id": "web-server",
  "variables": {
    "server_ip": "10.0.1.20",
    "allowed_sources": "0.0.0.0/0"
  },
  "dry_run": true
}'

api_call POST "/templates/instantiate" "$DRY_RUN_DATA" | jq '.preview[] | {name, direction, action, match}'

# 5. Create a multi-tier application setup
echo -e "\n${GREEN}5. Setting up multi-tier application:${NC}"

# Web tier
echo -e "${BLUE}Creating web server policy...${NC}"
WEB_DATA='{
  "template_id": "web-server",
  "variables": {
    "server_ip": "10.0.1.10",
    "allowed_sources": "0.0.0.0/0",
    "enable_ssh": true,
    "ssh_sources": "10.0.100.0/24"
  },
  "target_switch": "ls-web-tier"
}'
api_call POST "/templates/instantiate" "$WEB_DATA" | jq '{message, rules: .instance.rules | length}'

# Application tier
echo -e "${BLUE}Creating microservice policy...${NC}"
APP_DATA='{
  "template_id": "microservice",
  "variables": {
    "service_name": "api-gateway",
    "service_ip": "10.0.2.10",
    "service_port": 8080,
    "allowed_services": "10.0.1.10",
    "monitoring_subnet": "10.0.200.0/24"
  },
  "target_switch": "ls-app-tier"
}'
api_call POST "/templates/instantiate" "$APP_DATA" | jq '{message, rules: .instance.rules | length}'

# Database tier
echo -e "${BLUE}Creating database policy...${NC}"
DB_DATA='{
  "template_id": "database-server",
  "variables": {
    "db_ip": "10.0.3.10",
    "db_port": 5432,
    "app_subnet": "10.0.2.0/24",
    "backup_server": "10.0.100.50"
  },
  "target_switch": "ls-db-tier"
}'
api_call POST "/templates/instantiate" "$DB_DATA" | jq '{message, rules: .instance.rules | length}'

# 6. Import custom template
echo -e "\n${GREEN}6. Importing custom IoT device template:${NC}"
if [ -f "custom-template.json" ]; then
    CUSTOM_TEMPLATE=$(cat custom-template.json)
    api_call POST "/templates/import" "$CUSTOM_TEMPLATE" | jq '{template: .template.id, message}'
    
    # Use the imported template
    echo -e "${BLUE}Using imported IoT template...${NC}"
    IOT_DATA='{
      "template_id": "iot-device",
      "variables": {
        "device_ip": "10.0.50.100",
        "device_mac": "AA:BB:CC:DD:EE:FF",
        "mqtt_broker": "10.0.10.50",
        "allow_local_discovery": true
      },
      "target_switch": "ls-iot"
    }'
    api_call POST "/templates/instantiate" "$IOT_DATA" | jq '{message, rules: .instance.rules | length}'
fi

# 7. Export a template
echo -e "\n${GREEN}7. Exporting web-server template:${NC}"
curl -s -X GET \
    -H "Authorization: Bearer $TOKEN" \
    "$OVNCP_URL/api/v1/templates/web-server/export" \
    -o "web-server-template.json"
echo "Template exported to web-server-template.json"

# 8. Search templates by tag
echo -e "\n${GREEN}8. Searching templates by tag 'security':${NC}"
api_call GET "/templates?tag=security&tag=zone" | jq -r '.templates[] | "\(.id): \(.name)"'

# 9. Filter templates by category
echo -e "\n${GREEN}9. Listing Application category templates:${NC}"
api_call GET "/templates?category=Application" | jq -r '.templates[] | "\(.id): \(.name)"'

echo -e "\n${BLUE}Demo completed!${NC}"
echo "Templates provide a powerful way to standardize and automate network security policies."