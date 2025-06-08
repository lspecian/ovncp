package backup

import (
	"time"

	"github.com/lspecian/ovncp/internal/models"
)

// BackupFormat represents the format for backup files
type BackupFormat string

const (
	BackupFormatJSON BackupFormat = "json"
	BackupFormatYAML BackupFormat = "yaml"
)

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeSelective   BackupType = "selective"
)

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Type        BackupType        `json:"type" yaml:"type"`
	Format      BackupFormat      `json:"format" yaml:"format"`
	Version     string            `json:"version" yaml:"version"`
	CreatedAt   time.Time         `json:"created_at" yaml:"created_at"`
	CreatedBy   string            `json:"created_by" yaml:"created_by"`
	Size        int64             `json:"size" yaml:"size"`
	Checksum    string            `json:"checksum" yaml:"checksum"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Extra       map[string]string `json:"extra,omitempty" yaml:"extra,omitempty"`
}

// BackupData contains the actual backup data
type BackupData struct {
	Metadata         BackupMetadata                      `json:"metadata" yaml:"metadata"`
	LogicalSwitches  []*models.LogicalSwitch             `json:"logical_switches" yaml:"logical_switches"`
	LogicalRouters   []*models.LogicalRouter             `json:"logical_routers" yaml:"logical_routers"`
	LogicalPorts     []*LogicalPortWithSwitch            `json:"logical_ports" yaml:"logical_ports"`
	ACLs             []*ACLWithSwitch                    `json:"acls" yaml:"acls"`
	LoadBalancers    []*models.LoadBalancer              `json:"load_balancers,omitempty" yaml:"load_balancers,omitempty"`
	NATs             []*NATWithRouter                    `json:"nats,omitempty" yaml:"nats,omitempty"`
	DHCPOptions      []*models.DHCPOptions               `json:"dhcp_options,omitempty" yaml:"dhcp_options,omitempty"`
	QoSRules         []*models.QoS                       `json:"qos_rules,omitempty" yaml:"qos_rules,omitempty"`
	PortGroups       []*models.PortGroup                 `json:"port_groups,omitempty" yaml:"port_groups,omitempty"`
	AddressSets      []*models.AddressSet                `json:"address_sets,omitempty" yaml:"address_sets,omitempty"`
	ExternalIDs      map[string]map[string]string        `json:"external_ids,omitempty" yaml:"external_ids,omitempty"`
	Statistics       *BackupStatistics                   `json:"statistics,omitempty" yaml:"statistics,omitempty"`
}

// LogicalPortWithSwitch includes the switch information with the port
type LogicalPortWithSwitch struct {
	*models.LogicalSwitchPort
	SwitchID   string `json:"switch_id" yaml:"switch_id"`
	SwitchName string `json:"switch_name" yaml:"switch_name"`
}

// ACLWithSwitch includes the switch information with the ACL
type ACLWithSwitch struct {
	*models.ACL
	SwitchID   string `json:"switch_id" yaml:"switch_id"`
	SwitchName string `json:"switch_name" yaml:"switch_name"`
}

// NATWithRouter includes the router information with the NAT rule
type NATWithRouter struct {
	*models.NAT
	RouterID   string `json:"router_id" yaml:"router_id"`
	RouterName string `json:"router_name" yaml:"router_name"`
}

// BackupStatistics contains statistics about the backup
type BackupStatistics struct {
	TotalObjects      int            `json:"total_objects" yaml:"total_objects"`
	ObjectCounts      map[string]int `json:"object_counts" yaml:"object_counts"`
	ProcessingTime    time.Duration  `json:"processing_time" yaml:"processing_time"`
	CompressedSize    int64          `json:"compressed_size,omitempty" yaml:"compressed_size,omitempty"`
	UncompressedSize  int64          `json:"uncompressed_size" yaml:"uncompressed_size"`
}

// BackupOptions contains options for creating a backup
type BackupOptions struct {
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Type           BackupType        `json:"type"`
	Format         BackupFormat      `json:"format"`
	IncludeTags    []string          `json:"include_tags,omitempty"`
	ExcludeTags    []string          `json:"exclude_tags,omitempty"`
	ResourceFilter *ResourceFilter   `json:"resource_filter,omitempty"`
	Compress       bool              `json:"compress"`
	Encrypt        bool              `json:"encrypt"`
	EncryptionKey  string            `json:"-"` // Never serialize
	Tags           []string          `json:"tags,omitempty"`
	Extra          map[string]string `json:"extra,omitempty"`
}

// ResourceFilter specifies which resources to include in the backup
type ResourceFilter struct {
	Switches      []string `json:"switches,omitempty"`
	Routers       []string `json:"routers,omitempty"`
	PortGroups    []string `json:"port_groups,omitempty"`
	AddressSets   []string `json:"address_sets,omitempty"`
	IncludeACLs   bool     `json:"include_acls"`
	IncludePorts  bool     `json:"include_ports"`
	IncludeLBs    bool     `json:"include_lbs"`
	IncludeNATs   bool     `json:"include_nats"`
	IncludeDHCP   bool     `json:"include_dhcp"`
	IncludeQoS    bool     `json:"include_qos"`
}

// RestoreOptions contains options for restoring from a backup
type RestoreOptions struct {
	DryRun          bool              `json:"dry_run"`
	Force           bool              `json:"force"`
	SkipValidation  bool              `json:"skip_validation"`
	ResourceMapping map[string]string `json:"resource_mapping,omitempty"`
	ConflictPolicy  ConflictPolicy    `json:"conflict_policy"`
	RestoreFilter   *ResourceFilter   `json:"restore_filter,omitempty"`
	DecryptionKey   string            `json:"-"` // Never serialize
}

// ConflictPolicy defines how to handle conflicts during restore
type ConflictPolicy string

const (
	ConflictPolicySkip      ConflictPolicy = "skip"
	ConflictPolicyOverwrite ConflictPolicy = "overwrite"
	ConflictPolicyRename    ConflictPolicy = "rename"
	ConflictPolicyError     ConflictPolicy = "error"
)

// RestoreResult contains the result of a restore operation
type RestoreResult struct {
	Success         bool                     `json:"success"`
	RestoredCount   int                      `json:"restored_count"`
	SkippedCount    int                      `json:"skipped_count"`
	ErrorCount      int                      `json:"error_count"`
	Details         map[string]RestoreDetail `json:"details"`
	Errors          []string                 `json:"errors,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
	ProcessingTime  time.Duration            `json:"processing_time"`
}

// RestoreDetail contains details about restoring a specific resource type
type RestoreDetail struct {
	Total    int      `json:"total"`
	Restored int      `json:"restored"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

// BackupSchedule represents a scheduled backup configuration
type BackupSchedule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Schedule    string            `json:"schedule"` // Cron expression
	Options     BackupOptions     `json:"options"`
	Enabled     bool              `json:"enabled"`
	LastRun     *time.Time        `json:"last_run,omitempty"`
	NextRun     *time.Time        `json:"next_run,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Tags        []string          `json:"tags,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

// BackupStorage represents a storage backend for backups
type BackupStorage interface {
	// Store saves a backup to storage
	Store(backup *BackupData, options *BackupOptions) (string, error)
	
	// Retrieve gets a backup from storage
	Retrieve(backupID string) (*BackupData, error)
	
	// List returns all available backups
	List() ([]*BackupMetadata, error)
	
	// Delete removes a backup from storage
	Delete(backupID string) error
	
	// Exists checks if a backup exists
	Exists(backupID string) (bool, error)
}