#!/bin/bash

# OVN Control Platform Backup CLI
# A simple CLI tool for backup operations

set -euo pipefail

# Configuration
OVNCP_URL="${OVNCP_URL:-http://localhost:8080}"
OVNCP_TOKEN="${OVNCP_TOKEN:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_usage() {
    cat << EOF
OVN Control Platform Backup CLI

Usage: $0 <command> [options]

Commands:
    create      Create a new backup
    list        List all backups
    get         Get backup details
    restore     Restore from a backup
    delete      Delete a backup
    export      Export a backup
    import      Import a backup

Options:
    -h, --help  Show this help message

Environment variables:
    OVNCP_URL    API URL (default: http://localhost:8080)
    OVNCP_TOKEN  API Token (required)

Examples:
    $0 create --name "Daily Backup" --compress
    $0 list --tag production
    $0 restore --id <backup-id> --dry-run
    $0 export --id <backup-id> --format yaml > backup.yaml

EOF
}

error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Check token
check_auth() {
    if [ -z "$OVNCP_TOKEN" ]; then
        error "OVNCP_TOKEN environment variable is required"
    fi
}

# API call helper
api_call() {
    local method=$1
    local endpoint=$2
    local data=${3:-}
    
    local args=(
        -s
        -X "$method"
        -H "Authorization: Bearer $OVNCP_TOKEN"
        -H "Content-Type: application/json"
        "$OVNCP_URL/api/v1$endpoint"
    )
    
    if [ -n "$data" ]; then
        args+=(-d "$data")
    fi
    
    curl "${args[@]}"
}

# Commands
cmd_create() {
    local name=""
    local description=""
    local type="full"
    local compress=false
    local encrypt=false
    local tags=()
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --name)
                name="$2"
                shift 2
                ;;
            --description)
                description="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --compress)
                compress=true
                shift
                ;;
            --encrypt)
                encrypt=true
                shift
                ;;
            --tag)
                tags+=("\"$2\"")
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    if [ -z "$name" ]; then
        error "Backup name is required (--name)"
    fi
    
    # Build JSON
    local json="{\"name\": \"$name\", \"type\": \"$type\", \"compress\": $compress"
    
    if [ -n "$description" ]; then
        json+=", \"description\": \"$description\""
    fi
    
    if [ "$encrypt" = true ]; then
        read -s -p "Encryption password: " password
        echo
        json+=", \"encrypt\": true, \"encryption_key\": \"$password\""
    fi
    
    if [ ${#tags[@]} -gt 0 ]; then
        json+=", \"tags\": [${tags[*]}]"
    fi
    
    json+="}"
    
    info "Creating backup '$name'..."
    
    response=$(api_call POST /backups "$json")
    
    if echo "$response" | jq -e '.backup' > /dev/null 2>&1; then
        backup_id=$(echo "$response" | jq -r '.backup.id')
        success "Backup created successfully: $backup_id"
        echo "$response" | jq '.backup'
    else
        error "Failed to create backup: $(echo "$response" | jq -r '.error // "Unknown error"')"
    fi
}

cmd_list() {
    local tag=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --tag)
                tag="$2"
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    local endpoint="/backups"
    if [ -n "$tag" ]; then
        endpoint+="?tag=$tag"
    fi
    
    info "Listing backups..."
    
    response=$(api_call GET "$endpoint")
    
    if echo "$response" | jq -e '.backups' > /dev/null 2>&1; then
        count=$(echo "$response" | jq '.total')
        success "Found $count backup(s)"
        echo
        echo "$response" | jq -r '.backups[] | "\(.id) | \(.name) | \(.created_at) | \(.size) bytes | Tags: \(.tags | join(", "))"' | \
            column -t -s '|' -N "ID,NAME,CREATED,SIZE,TAGS"
    else
        error "Failed to list backups"
    fi
}

cmd_get() {
    local id=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --id)
                id="$2"
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    if [ -z "$id" ]; then
        error "Backup ID is required (--id)"
    fi
    
    info "Getting backup details..."
    
    response=$(api_call GET "/backups/$id")
    
    if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        success "Backup details:"
        echo "$response" | jq '.'
    else
        error "Backup not found"
    fi
}

cmd_restore() {
    local id=""
    local dry_run=false
    local policy="skip"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --id)
                id="$2"
                shift 2
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --policy)
                policy="$2"
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    if [ -z "$id" ]; then
        error "Backup ID is required (--id)"
    fi
    
    local json="{\"dry_run\": $dry_run, \"conflict_policy\": \"$policy\""
    
    # Check if backup is encrypted
    backup_info=$(api_call GET "/backups/$id")
    if echo "$backup_info" | jq -e '.extra.encrypted' > /dev/null 2>&1; then
        read -s -p "Decryption password: " password
        echo
        json+=", \"decryption_key\": \"$password\""
    fi
    
    json+="}"
    
    if [ "$dry_run" = true ]; then
        info "Performing dry-run restore..."
    else
        warning "Restoring backup $id (conflict policy: $policy)"
        read -p "Are you sure? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Restore cancelled"
            exit 0
        fi
    fi
    
    response=$(api_call POST "/backups/$id/restore" "$json")
    
    if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
        if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
            success "Restore completed successfully"
        else
            warning "Restore completed with errors"
        fi
        
        echo
        echo "Summary:"
        echo "  Restored: $(echo "$response" | jq -r '.restored_count')"
        echo "  Skipped:  $(echo "$response" | jq -r '.skipped_count')"
        echo "  Errors:   $(echo "$response" | jq -r '.error_count')"
        echo
        echo "Details:"
        echo "$response" | jq '.details'
        
        if echo "$response" | jq -e '.errors[]' > /dev/null 2>&1; then
            echo
            echo "Errors:"
            echo "$response" | jq -r '.errors[]'
        fi
    else
        error "Failed to restore backup"
    fi
}

cmd_delete() {
    local id=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --id)
                id="$2"
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    if [ -z "$id" ]; then
        error "Backup ID is required (--id)"
    fi
    
    warning "Deleting backup $id"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        info "Delete cancelled"
        exit 0
    fi
    
    response=$(api_call DELETE "/backups/$id")
    
    if echo "$response" | jq -e '.message' > /dev/null 2>&1; then
        success "Backup deleted successfully"
    else
        error "Failed to delete backup"
    fi
}

cmd_export() {
    local id=""
    local format="json"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --id)
                id="$2"
                shift 2
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    if [ -z "$id" ]; then
        error "Backup ID is required (--id)"
    fi
    
    info "Exporting backup..." >&2
    
    curl -s -X GET \
        -H "Authorization: Bearer $OVNCP_TOKEN" \
        "$OVNCP_URL/api/v1/backups/$id/export?format=$format"
}

cmd_import() {
    local file=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --file)
                file="$2"
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    if [ -z "$file" ]; then
        error "Backup file is required (--file)"
    fi
    
    if [ ! -f "$file" ]; then
        error "File not found: $file"
    fi
    
    info "Importing backup from $file..."
    
    response=$(curl -s -X POST \
        -H "Authorization: Bearer $OVNCP_TOKEN" \
        -F "file=@$file" \
        "$OVNCP_URL/api/v1/backups/import")
    
    if echo "$response" | jq -e '.backup' > /dev/null 2>&1; then
        backup_id=$(echo "$response" | jq -r '.backup.id')
        success "Backup imported successfully: $backup_id"
        echo "$response" | jq '.backup'
    else
        error "Failed to import backup"
    fi
}

# Main
check_auth

if [ $# -eq 0 ]; then
    print_usage
    exit 1
fi

command=$1
shift

case $command in
    create)
        cmd_create "$@"
        ;;
    list)
        cmd_list "$@"
        ;;
    get)
        cmd_get "$@"
        ;;
    restore)
        cmd_restore "$@"
        ;;
    delete)
        cmd_delete "$@"
        ;;
    export)
        cmd_export "$@"
        ;;
    import)
        cmd_import "$@"
        ;;
    -h|--help|help)
        print_usage
        ;;
    *)
        error "Unknown command: $command"
        ;;
esac