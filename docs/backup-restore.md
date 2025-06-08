# Backup and Restore

The OVN Control Platform provides comprehensive backup and restore functionality to protect your network configurations and enable disaster recovery.

## Overview

The backup system allows you to:
- Create full or selective backups of OVN configurations
- Restore configurations with conflict resolution
- Schedule automated backups
- Export/import backups for archival
- Encrypt sensitive backup data
- Compress backups to save storage

## Backup Types

### 1. Full Backup

Captures the entire OVN configuration including:
- Logical switches
- Logical routers  
- Ports and port configurations
- ACLs (Access Control Lists)
- Load balancers
- NAT rules
- DHCP options
- QoS rules
- Port groups
- Address sets

```bash
curl -X POST $OVNCP_URL/api/v1/backups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Full Backup - Production",
    "description": "Complete backup before maintenance",
    "type": "full",
    "compress": true,
    "tags": ["production", "pre-maintenance"]
  }'
```

### 2. Selective Backup

Backs up only specified resources:

```bash
curl -X POST $OVNCP_URL/api/v1/backups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Web Tier Backup",
    "type": "selective",
    "resource_filter": {
      "switches": ["ls-web-tier", "ls-app-tier"],
      "routers": ["lr-main"],
      "include_acls": true,
      "include_ports": true
    }
  }'
```

### 3. Incremental Backup (Coming Soon)

Captures only changes since the last backup, reducing storage and processing time.

## API Endpoints

### Create Backup

```http
POST /api/v1/backups
```

Request:
```json
{
  "name": "Daily Backup",
  "description": "Automated daily backup",
  "type": "full",
  "format": "json",
  "compress": true,
  "encrypt": true,
  "encryption_key": "your-strong-password",
  "tags": ["daily", "automated"],
  "resource_filter": {
    "switches": ["ls1", "ls2"],
    "include_acls": true
  }
}
```

Response:
```json
{
  "backup": {
    "id": "b1234567-89ab-cdef-0123-456789abcdef",
    "name": "Daily Backup",
    "type": "full",
    "format": "json",
    "created_at": "2024-01-15T10:30:00Z",
    "size": 1048576,
    "checksum": "sha256:abcdef...",
    "tags": ["daily", "automated"]
  },
  "message": "Backup created successfully"
}
```

### List Backups

```http
GET /api/v1/backups?tag=production
```

Response:
```json
{
  "backups": [
    {
      "id": "b1234567-89ab-cdef-0123-456789abcdef",
      "name": "Production Backup",
      "created_at": "2024-01-15T10:30:00Z",
      "size": 2097152,
      "tags": ["production"]
    }
  ],
  "total": 1
}
```

### Restore Backup

```http
POST /api/v1/backups/:id/restore
```

Request:
```json
{
  "dry_run": false,
  "conflict_policy": "skip",
  "resource_mapping": {
    "old-switch-id": "new-switch-id"
  },
  "decryption_key": "your-strong-password"
}
```

Response:
```json
{
  "success": true,
  "restored_count": 45,
  "skipped_count": 3,
  "error_count": 0,
  "details": {
    "switches": {
      "total": 10,
      "restored": 10,
      "skipped": 0,
      "failed": 0
    },
    "acls": {
      "total": 35,
      "restored": 32,
      "skipped": 3,
      "failed": 0
    }
  },
  "processing_time": "1.234s"
}
```

### Export Backup

```http
GET /api/v1/backups/:id/export?format=yaml
```

Downloads the backup file in the specified format (json or yaml).

### Import Backup

```http
POST /api/v1/backups/import
```

Upload a backup file or provide JSON data:

```bash
# File upload
curl -X POST $OVNCP_URL/api/v1/backups/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@backup.json"

# JSON data
curl -X POST $OVNCP_URL/api/v1/backups/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "data": "{...backup data...}"
  }'
```

## Restore Options

### Conflict Policies

When restoring, you can specify how to handle existing resources:

- **skip**: Skip resources that already exist (default)
- **overwrite**: Delete existing resources and recreate
- **rename**: Rename restored resources with timestamp suffix
- **error**: Fail on any conflict

### Dry Run Mode

Test restore without making changes:

```bash
curl -X POST $OVNCP_URL/api/v1/backups/$BACKUP_ID/restore \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "dry_run": true,
    "conflict_policy": "overwrite"
  }'
```

### Resource Mapping

Map old resource IDs to new ones during restore:

```json
{
  "resource_mapping": {
    "old-switch-uuid": "new-switch-uuid",
    "old-router-uuid": "new-router-uuid"
  }
}
```

### Selective Restore

Restore only specific resources:

```json
{
  "restore_filter": {
    "switches": ["ls-web", "ls-app"],
    "include_acls": true,
    "include_ports": false
  }
}
```

## Backup Storage

### File Storage

By default, backups are stored in the local filesystem:

```bash
# Set backup directory
export BACKUP_PATH=/var/lib/ovncp/backups

# Directory structure
/var/lib/ovncp/backups/
├── Daily_Backup-20240115-103000.json.gz
├── Daily_Backup-20240115-103000.json.gz.meta
├── Production_Backup-20240114-200000.json.gz.enc
└── Production_Backup-20240114-200000.json.gz.enc.meta
```

### Storage Formats

- **.json**: Plain JSON format
- **.yaml**: YAML format
- **.gz**: Gzip compressed
- **.enc**: AES-256 encrypted
- **.meta**: Metadata file

## Security

### Encryption

Backups can be encrypted using AES-256-GCM:

```bash
curl -X POST $OVNCP_URL/api/v1/backups \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Secure Backup",
    "encrypt": true,
    "encryption_key": "my-strong-password-32-chars-long"
  }'
```

**Important**: 
- Store encryption keys securely
- Use strong passwords (32+ characters recommended)
- Keys are never stored with backups

### Access Control

Backup operations require specific permissions:

- `backups:read` - List and download backups
- `backups:write` - Create backups
- `backups:delete` - Delete backups
- `backups:restore` - Restore from backups (also requires `admin`)

## Automation

### Scheduled Backups

Create automated backups using cron or systemd timers:

```bash
#!/bin/bash
# daily-backup.sh

OVNCP_URL="https://ovncp.example.com"
TOKEN="your-api-token"

# Create daily backup
RESPONSE=$(curl -s -X POST $OVNCP_URL/api/v1/backups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Automated Daily Backup",
    "type": "full",
    "compress": true,
    "tags": ["daily", "automated"]
  }')

BACKUP_ID=$(echo $RESPONSE | jq -r '.backup.id')
echo "Created backup: $BACKUP_ID"

# Clean up old backups (keep last 7 days)
CUTOFF_DATE=$(date -d '7 days ago' --iso-8601)
curl -s $OVNCP_URL/api/v1/backups \
  -H "Authorization: Bearer $TOKEN" | \
  jq -r ".backups[] | select(.created_at < \"$CUTOFF_DATE\") | .id" | \
  while read id; do
    echo "Deleting old backup: $id"
    curl -X DELETE $OVNCP_URL/api/v1/backups/$id \
      -H "Authorization: Bearer $TOKEN"
  done
```

### Backup Rotation

Implement backup rotation policies:

```python
import requests
from datetime import datetime, timedelta

class BackupRotation:
    def __init__(self, api_url, token):
        self.api_url = api_url
        self.headers = {'Authorization': f'Bearer {token}'}
    
    def rotate_backups(self, keep_daily=7, keep_weekly=4, keep_monthly=12):
        """Implement grandfather-father-son rotation"""
        
        # Get all backups
        response = requests.get(f"{self.api_url}/backups", headers=self.headers)
        backups = response.json()['backups']
        
        # Sort by creation date
        backups.sort(key=lambda x: x['created_at'], reverse=True)
        
        now = datetime.now()
        daily_cutoff = now - timedelta(days=keep_daily)
        weekly_cutoff = now - timedelta(weeks=keep_weekly)
        monthly_cutoff = now - timedelta(days=keep_monthly * 30)
        
        to_keep = set()
        to_delete = []
        
        for backup in backups:
            created = datetime.fromisoformat(backup['created_at'].replace('Z', '+00:00'))
            
            # Keep all recent daily backups
            if created > daily_cutoff:
                to_keep.add(backup['id'])
            # Keep weekly backups (Sunday)
            elif created > weekly_cutoff and created.weekday() == 6:
                to_keep.add(backup['id'])
            # Keep monthly backups (1st of month)
            elif created > monthly_cutoff and created.day == 1:
                to_keep.add(backup['id'])
            else:
                to_delete.append(backup['id'])
        
        # Delete old backups
        for backup_id in to_delete:
            requests.delete(f"{self.api_url}/backups/{backup_id}", headers=self.headers)
            print(f"Deleted backup: {backup_id}")
        
        print(f"Kept {len(to_keep)} backups, deleted {len(to_delete)}")
```

## Best Practices

### 1. Regular Backups

- **Production**: Daily full backups, hourly selective backups
- **Staging**: Weekly full backups
- **Development**: On-demand backups before major changes

### 2. Testing Restores

Regularly test restore procedures:

```bash
# 1. Create test environment
# 2. Perform dry-run restore
curl -X POST $OVNCP_URL/api/v1/backups/$BACKUP_ID/restore \
  -d '{"dry_run": true}'

# 3. Review what would be restored
# 4. Perform actual restore in test environment
# 5. Validate restored configuration
```

### 3. Backup Naming

Use descriptive names with timestamps:

```json
{
  "name": "prod-pre-upgrade-v2.5.0",
  "description": "Production backup before upgrading to v2.5.0",
  "tags": ["production", "pre-upgrade", "v2.5.0"]
}
```

### 4. Monitoring

Monitor backup operations:

- Set up alerts for backup failures
- Track backup sizes and growth
- Monitor restore times
- Validate backup integrity

### 5. Documentation

Document your backup procedures:

- Backup schedules
- Retention policies
- Restore procedures
- Contact information
- Encryption key management

## Disaster Recovery

### Recovery Plan

1. **Identify the issue**
   - Configuration corruption
   - Accidental deletion
   - System failure

2. **Select appropriate backup**
   ```bash
   # List recent backups
   curl $OVNCP_URL/api/v1/backups?tag=production
   ```

3. **Perform dry-run restore**
   ```bash
   curl -X POST $OVNCP_URL/api/v1/backups/$BACKUP_ID/restore \
     -d '{"dry_run": true}'
   ```

4. **Execute restore**
   ```bash
   curl -X POST $OVNCP_URL/api/v1/backups/$BACKUP_ID/restore \
     -d '{"conflict_policy": "overwrite"}'
   ```

5. **Validate restoration**
   - Check network connectivity
   - Verify ACL rules
   - Test critical paths

### RTO and RPO

- **Recovery Time Objective (RTO)**: < 15 minutes
- **Recovery Point Objective (RPO)**: < 1 hour

Achieve these targets by:
- Automating backup procedures
- Practicing restore procedures
- Maintaining backup documentation
- Using incremental backups (when available)

## Troubleshooting

### Common Issues

**"Backup not found"**
- Check backup ID is correct
- Verify backup hasn't been deleted
- Ensure proper permissions

**"Decryption failed"**
- Verify encryption key is correct
- Check key hasn't been changed
- Ensure backup is actually encrypted

**"Restore conflicts"**
- Review conflict policy
- Check existing resources
- Use dry-run to preview changes

**"Insufficient permissions"**
- Verify user has `backups:restore` permission
- Admin permission required for restore
- Check API token is valid

### Debug Mode

Enable detailed logging for troubleshooting:

```bash
curl -X POST $OVNCP_URL/api/v1/backups/$BACKUP_ID/restore \
  -H "X-Debug: true" \
  -d '{"dry_run": true}'
```

## Examples

### Complete Disaster Recovery

```bash
#!/bin/bash
# disaster-recovery.sh

# 1. Find latest production backup
LATEST_BACKUP=$(curl -s $OVNCP_URL/api/v1/backups?tag=production | \
  jq -r '.backups[0].id')

echo "Latest backup: $LATEST_BACKUP"

# 2. Validate backup
curl -X POST $OVNCP_URL/api/v1/backups/validate \
  -d "{\"backup_id\": \"$LATEST_BACKUP\"}"

# 3. Perform dry-run
echo "Performing dry-run restore..."
DRY_RUN=$(curl -s -X POST $OVNCP_URL/api/v1/backups/$LATEST_BACKUP/restore \
  -d '{"dry_run": true}')

echo "Would restore:"
echo $DRY_RUN | jq '.details'

# 4. Confirm and restore
read -p "Proceed with restore? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  echo "Restoring..."
  curl -X POST $OVNCP_URL/api/v1/backups/$LATEST_BACKUP/restore \
    -d '{"conflict_policy": "overwrite", "force": true}'
fi
```

### Migrate Between Environments

```python
import requests
import json

def migrate_config(source_url, source_token, dest_url, dest_token):
    """Migrate configuration between environments"""
    
    # 1. Create backup in source
    print("Creating backup in source environment...")
    response = requests.post(
        f"{source_url}/api/v1/backups",
        headers={"Authorization": f"Bearer {source_token}"},
        json={
            "name": "Migration Export",
            "type": "full",
            "compress": True
        }
    )
    backup_id = response.json()['backup']['id']
    
    # 2. Export backup
    print(f"Exporting backup {backup_id}...")
    response = requests.get(
        f"{source_url}/api/v1/backups/{backup_id}/export",
        headers={"Authorization": f"Bearer {source_token}"}
    )
    backup_data = response.text
    
    # 3. Import to destination
    print("Importing to destination environment...")
    response = requests.post(
        f"{dest_url}/api/v1/backups/import",
        headers={"Authorization": f"Bearer {dest_token}"},
        json={
            "format": "json",
            "data": backup_data
        }
    )
    imported_id = response.json()['backup']['id']
    
    # 4. Restore in destination
    print(f"Restoring backup {imported_id}...")
    response = requests.post(
        f"{dest_url}/api/v1/backups/{imported_id}/restore",
        headers={"Authorization": f"Bearer {dest_token}"},
        json={
            "conflict_policy": "skip"
        }
    )
    
    result = response.json()
    print(f"Migration complete: {result['restored_count']} resources restored")
    
    return result

# Usage
migrate_config(
    "https://staging.ovncp.example.com",
    "staging-token",
    "https://prod.ovncp.example.com", 
    "prod-token"
)
```