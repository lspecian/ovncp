package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// BackupService handles backup and restore operations
type BackupService struct {
	ovnService services.OVNServiceInterface
	storage    BackupStorage
	logger     *zap.Logger
}

// NewBackupService creates a new backup service
func NewBackupService(ovnService services.OVNServiceInterface, storage BackupStorage, logger *zap.Logger) *BackupService {
	return &BackupService{
		ovnService: ovnService,
		storage:    storage,
		logger:     logger,
	}
}

// CreateBackup creates a backup of OVN configuration
func (s *BackupService) CreateBackup(ctx context.Context, options *BackupOptions) (*BackupMetadata, error) {
	startTime := time.Now()
	
	// Set defaults
	if options.Format == "" {
		options.Format = BackupFormatJSON
	}
	if options.Type == "" {
		options.Type = BackupTypeFull
	}
	
	// Create backup data structure
	backupData := &BackupData{
		Metadata: BackupMetadata{
			ID:          uuid.New().String(),
			Name:        options.Name,
			Description: options.Description,
			Type:        options.Type,
			Format:      options.Format,
			Version:     "1.0",
			CreatedAt:   time.Now(),
			CreatedBy:   "system", // TODO: Get from context
			Tags:        options.Tags,
			Extra:       options.Extra,
		},
		Statistics: &BackupStatistics{
			ObjectCounts: make(map[string]int),
		},
	}

	// Collect data based on backup type
	switch options.Type {
	case BackupTypeFull:
		if err := s.collectFullBackup(ctx, backupData, options); err != nil {
			return nil, fmt.Errorf("failed to collect full backup: %w", err)
		}
	case BackupTypeSelective:
		if err := s.collectSelectiveBackup(ctx, backupData, options); err != nil {
			return nil, fmt.Errorf("failed to collect selective backup: %w", err)
		}
	case BackupTypeIncremental:
		// TODO: Implement incremental backup
		return nil, fmt.Errorf("incremental backup not yet implemented")
	}

	// Calculate statistics
	backupData.Statistics.ProcessingTime = time.Since(startTime)
	backupData.Statistics.TotalObjects = s.calculateTotalObjects(backupData)

	// Store the backup
	backupID, err := s.storage.Store(backupData, options)
	if err != nil {
		return nil, fmt.Errorf("failed to store backup: %w", err)
	}

	backupData.Metadata.ID = backupID
	s.logger.Info("Backup created successfully",
		zap.String("backup_id", backupID),
		zap.String("name", options.Name),
		zap.Int("total_objects", backupData.Statistics.TotalObjects),
		zap.Duration("processing_time", backupData.Statistics.ProcessingTime))

	return &backupData.Metadata, nil
}

// collectFullBackup collects all OVN resources
func (s *BackupService) collectFullBackup(ctx context.Context, backup *BackupData, options *BackupOptions) error {
	// Collect logical switches
	switches, err := s.ovnService.ListLogicalSwitches(ctx)
	if err != nil {
		return fmt.Errorf("failed to list logical switches: %w", err)
	}
	backup.LogicalSwitches = switches
	backup.Statistics.ObjectCounts["switches"] = len(switches)

	// Collect logical routers
	routers, err := s.ovnService.ListLogicalRouters(ctx)
	if err != nil {
		return fmt.Errorf("failed to list logical routers: %w", err)
	}
	backup.LogicalRouters = routers
	backup.Statistics.ObjectCounts["routers"] = len(routers)

	// Collect ports for each switch
	backup.LogicalPorts = []*LogicalPortWithSwitch{}
	for _, sw := range switches {
		ports, err := s.ovnService.ListPorts(ctx, sw.UUID)
		if err != nil {
			s.logger.Warn("Failed to list ports for switch",
				zap.String("switch", sw.Name),
				zap.Error(err))
			continue
		}
		
		for _, port := range ports {
			backup.LogicalPorts = append(backup.LogicalPorts, &LogicalPortWithSwitch{
				LogicalSwitchPort: port,
				SwitchID:          sw.UUID,
				SwitchName:        sw.Name,
			})
		}
	}
	backup.Statistics.ObjectCounts["ports"] = len(backup.LogicalPorts)

	// Collect ACLs for each switch
	backup.ACLs = []*ACLWithSwitch{}
	for _, sw := range switches {
		acls, err := s.ovnService.ListACLs(ctx, sw.UUID)
		if err != nil {
			s.logger.Warn("Failed to list ACLs for switch",
				zap.String("switch", sw.Name),
				zap.Error(err))
			continue
		}
		
		for _, acl := range acls {
			backup.ACLs = append(backup.ACLs, &ACLWithSwitch{
				ACL:        acl,
				SwitchID:   sw.UUID,
				SwitchName: sw.Name,
			})
		}
	}
	backup.Statistics.ObjectCounts["acls"] = len(backup.ACLs)

	// TODO: Collect other resources (LoadBalancers, NATs, etc.)

	return nil
}

// collectSelectiveBackup collects only specified resources
func (s *BackupService) collectSelectiveBackup(ctx context.Context, backup *BackupData, options *BackupOptions) error {
	if options.ResourceFilter == nil {
		return fmt.Errorf("resource filter required for selective backup")
	}

	filter := options.ResourceFilter

	// Collect specified switches
	if len(filter.Switches) > 0 {
		backup.LogicalSwitches = []*models.LogicalSwitch{}
		for _, switchID := range filter.Switches {
			sw, err := s.ovnService.GetLogicalSwitch(ctx, switchID)
			if err != nil {
				s.logger.Warn("Failed to get switch",
					zap.String("switch_id", switchID),
					zap.Error(err))
				continue
			}
			backup.LogicalSwitches = append(backup.LogicalSwitches, sw)

			// Collect ports if requested
			if filter.IncludePorts {
				ports, err := s.ovnService.ListPorts(ctx, sw.UUID)
				if err != nil {
					s.logger.Warn("Failed to list ports",
						zap.String("switch", sw.Name),
						zap.Error(err))
					continue
				}
				
				for _, port := range ports {
					backup.LogicalPorts = append(backup.LogicalPorts, &LogicalPortWithSwitch{
						LogicalSwitchPort: port,
						SwitchID:          sw.UUID,
						SwitchName:        sw.Name,
					})
				}
			}

			// Collect ACLs if requested
			if filter.IncludeACLs {
				acls, err := s.ovnService.ListACLs(ctx, sw.UUID)
				if err != nil {
					s.logger.Warn("Failed to list ACLs",
						zap.String("switch", sw.Name),
						zap.Error(err))
					continue
				}
				
				for _, acl := range acls {
					backup.ACLs = append(backup.ACLs, &ACLWithSwitch{
						ACL:        acl,
						SwitchID:   sw.UUID,
						SwitchName: sw.Name,
					})
				}
			}
		}
		backup.Statistics.ObjectCounts["switches"] = len(backup.LogicalSwitches)
		backup.Statistics.ObjectCounts["ports"] = len(backup.LogicalPorts)
		backup.Statistics.ObjectCounts["acls"] = len(backup.ACLs)
	}

	// Collect specified routers
	if len(filter.Routers) > 0 {
		backup.LogicalRouters = []*models.LogicalRouter{}
		for _, routerID := range filter.Routers {
			router, err := s.ovnService.GetLogicalRouter(ctx, routerID)
			if err != nil {
				s.logger.Warn("Failed to get router",
					zap.String("router_id", routerID),
					zap.Error(err))
				continue
			}
			backup.LogicalRouters = append(backup.LogicalRouters, router)
		}
		backup.Statistics.ObjectCounts["routers"] = len(backup.LogicalRouters)
	}

	return nil
}

// RestoreBackup restores OVN configuration from a backup
func (s *BackupService) RestoreBackup(ctx context.Context, backupID string, options *RestoreOptions) (*RestoreResult, error) {
	startTime := time.Now()

	// Retrieve backup
	backupData, err := s.storage.Retrieve(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve backup: %w", err)
	}

	result := &RestoreResult{
		Success: true,
		Details: make(map[string]RestoreDetail),
	}

	// Validate backup if required
	if !options.SkipValidation {
		if err := s.validateBackup(backupData); err != nil {
			return nil, fmt.Errorf("backup validation failed: %w", err)
		}
	}

	// Dry run mode - just validate and return what would be restored
	if options.DryRun {
		return s.dryRunRestore(ctx, backupData, options)
	}

	// Restore logical switches
	if err := s.restoreSwitches(ctx, backupData, options, result); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to restore switches: %v", err))
	}

	// Restore logical routers
	if err := s.restoreRouters(ctx, backupData, options, result); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to restore routers: %v", err))
	}

	// Restore ports (must be after switches)
	if err := s.restorePorts(ctx, backupData, options, result); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to restore ports: %v", err))
	}

	// Restore ACLs (must be after switches)
	if err := s.restoreACLs(ctx, backupData, options, result); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to restore ACLs: %v", err))
	}

	result.ProcessingTime = time.Since(startTime)

	s.logger.Info("Restore completed",
		zap.String("backup_id", backupID),
		zap.Bool("success", result.Success),
		zap.Int("restored", result.RestoredCount),
		zap.Int("skipped", result.SkippedCount),
		zap.Int("errors", result.ErrorCount),
		zap.Duration("processing_time", result.ProcessingTime))

	return result, nil
}

// restoreSwitches restores logical switches
func (s *BackupService) restoreSwitches(ctx context.Context, backup *BackupData, options *RestoreOptions, result *RestoreResult) error {
	detail := RestoreDetail{
		Total: len(backup.LogicalSwitches),
	}

	for _, sw := range backup.LogicalSwitches {
		// Check if switch already exists
		existing, err := s.ovnService.GetLogicalSwitch(ctx, sw.Name)
		if err == nil && existing != nil {
			// Handle conflict
			switch options.ConflictPolicy {
			case ConflictPolicySkip:
				detail.Skipped++
				result.SkippedCount++
				continue
			case ConflictPolicyOverwrite:
				if err := s.ovnService.DeleteLogicalSwitch(ctx, existing.UUID); err != nil {
					detail.Failed++
					detail.Errors = append(detail.Errors, fmt.Sprintf("Failed to delete existing switch %s: %v", sw.Name, err))
					continue
				}
			case ConflictPolicyRename:
				sw.Name = fmt.Sprintf("%s_restored_%d", sw.Name, time.Now().Unix())
			case ConflictPolicyError:
				detail.Failed++
				detail.Errors = append(detail.Errors, fmt.Sprintf("Switch %s already exists", sw.Name))
				continue
			}
		}

		// Create the switch
		_, err = s.ovnService.CreateLogicalSwitch(ctx, sw)
		if err != nil {
			detail.Failed++
			detail.Errors = append(detail.Errors, fmt.Sprintf("Failed to create switch %s: %v", sw.Name, err))
			result.ErrorCount++
		} else {
			detail.Restored++
			result.RestoredCount++
		}
	}

	result.Details["switches"] = detail
	return nil
}

// restoreRouters restores logical routers
func (s *BackupService) restoreRouters(ctx context.Context, backup *BackupData, options *RestoreOptions, result *RestoreResult) error {
	detail := RestoreDetail{
		Total: len(backup.LogicalRouters),
	}

	for _, router := range backup.LogicalRouters {
		// Check if router already exists
		existing, err := s.ovnService.GetLogicalRouter(ctx, router.Name)
		if err == nil && existing != nil {
			// Handle conflict
			switch options.ConflictPolicy {
			case ConflictPolicySkip:
				detail.Skipped++
				result.SkippedCount++
				continue
			case ConflictPolicyOverwrite:
				if err := s.ovnService.DeleteLogicalRouter(ctx, existing.UUID); err != nil {
					detail.Failed++
					detail.Errors = append(detail.Errors, fmt.Sprintf("Failed to delete existing router %s: %v", router.Name, err))
					continue
				}
			case ConflictPolicyRename:
				router.Name = fmt.Sprintf("%s_restored_%d", router.Name, time.Now().Unix())
			case ConflictPolicyError:
				detail.Failed++
				detail.Errors = append(detail.Errors, fmt.Sprintf("Router %s already exists", router.Name))
				continue
			}
		}

		// Create the router
		_, err = s.ovnService.CreateLogicalRouter(ctx, router)
		if err != nil {
			detail.Failed++
			detail.Errors = append(detail.Errors, fmt.Sprintf("Failed to create router %s: %v", router.Name, err))
			result.ErrorCount++
		} else {
			detail.Restored++
			result.RestoredCount++
		}
	}

	result.Details["routers"] = detail
	return nil
}

// restorePorts restores logical switch ports
func (s *BackupService) restorePorts(ctx context.Context, backup *BackupData, options *RestoreOptions, result *RestoreResult) error {
	detail := RestoreDetail{
		Total: len(backup.LogicalPorts),
	}

	for _, portWithSwitch := range backup.LogicalPorts {
		port := portWithSwitch.LogicalSwitchPort
		
		// Find the switch (it might have been renamed)
		switchID := portWithSwitch.SwitchID
		if options.ResourceMapping != nil {
			if mappedID, ok := options.ResourceMapping[switchID]; ok {
				switchID = mappedID
			}
		}

		// Create the port
		_, err := s.ovnService.CreatePort(ctx, switchID, port)
		if err != nil {
			detail.Failed++
			detail.Errors = append(detail.Errors, fmt.Sprintf("Failed to create port %s: %v", port.Name, err))
			result.ErrorCount++
		} else {
			detail.Restored++
			result.RestoredCount++
		}
	}

	result.Details["ports"] = detail
	return nil
}

// restoreACLs restores ACLs
func (s *BackupService) restoreACLs(ctx context.Context, backup *BackupData, options *RestoreOptions, result *RestoreResult) error {
	detail := RestoreDetail{
		Total: len(backup.ACLs),
	}

	for _, aclWithSwitch := range backup.ACLs {
		acl := aclWithSwitch.ACL
		
		// Find the switch (it might have been renamed)
		switchID := aclWithSwitch.SwitchID
		if options.ResourceMapping != nil {
			if mappedID, ok := options.ResourceMapping[switchID]; ok {
				switchID = mappedID
			}
		}

		// Create the ACL
		_, err := s.ovnService.CreateACL(ctx, switchID, acl)
		if err != nil {
			detail.Failed++
			detail.Errors = append(detail.Errors, fmt.Sprintf("Failed to create ACL %s: %v", acl.Name, err))
			result.ErrorCount++
		} else {
			detail.Restored++
			result.RestoredCount++
		}
	}

	result.Details["acls"] = detail
	return nil
}

// validateBackup validates backup data integrity
func (s *BackupService) validateBackup(backup *BackupData) error {
	// Validate metadata
	if backup.Metadata.ID == "" {
		return fmt.Errorf("backup ID is empty")
	}
	if backup.Metadata.Version == "" {
		return fmt.Errorf("backup version is empty")
	}

	// TODO: Add more validation
	// - Check resource references
	// - Validate data integrity
	// - Check version compatibility

	return nil
}

// dryRunRestore simulates a restore operation
func (s *BackupService) dryRunRestore(ctx context.Context, backup *BackupData, options *RestoreOptions) (*RestoreResult, error) {
	result := &RestoreResult{
		Success: true,
		Details: make(map[string]RestoreDetail),
	}

	// Check switches
	detail := RestoreDetail{Total: len(backup.LogicalSwitches)}
	for _, sw := range backup.LogicalSwitches {
		existing, _ := s.ovnService.GetLogicalSwitch(ctx, sw.Name)
		if existing != nil {
			switch options.ConflictPolicy {
			case ConflictPolicySkip:
				detail.Skipped++
			case ConflictPolicyOverwrite, ConflictPolicyRename:
				detail.Restored++
			case ConflictPolicyError:
				detail.Failed++
			}
		} else {
			detail.Restored++
		}
	}
	result.Details["switches"] = detail

	// Check routers
	detail = RestoreDetail{Total: len(backup.LogicalRouters)}
	for _, router := range backup.LogicalRouters {
		existing, _ := s.ovnService.GetLogicalRouter(ctx, router.Name)
		if existing != nil {
			switch options.ConflictPolicy {
			case ConflictPolicySkip:
				detail.Skipped++
			case ConflictPolicyOverwrite, ConflictPolicyRename:
				detail.Restored++
			case ConflictPolicyError:
				detail.Failed++
			}
		} else {
			detail.Restored++
		}
	}
	result.Details["routers"] = detail

	// Ports and ACLs would be restored based on switch availability
	result.Details["ports"] = RestoreDetail{
		Total:    len(backup.LogicalPorts),
		Restored: len(backup.LogicalPorts), // Assume all would be restored in dry run
	}
	result.Details["acls"] = RestoreDetail{
		Total:    len(backup.ACLs),
		Restored: len(backup.ACLs), // Assume all would be restored in dry run
	}

	// Calculate totals
	for _, detail := range result.Details {
		result.RestoredCount += detail.Restored
		result.SkippedCount += detail.Skipped
		result.ErrorCount += detail.Failed
	}

	return result, nil
}

// calculateTotalObjects calculates the total number of objects in a backup
func (s *BackupService) calculateTotalObjects(backup *BackupData) int {
	total := 0
	total += len(backup.LogicalSwitches)
	total += len(backup.LogicalRouters)
	total += len(backup.LogicalPorts)
	total += len(backup.ACLs)
	total += len(backup.LoadBalancers)
	total += len(backup.NATs)
	total += len(backup.DHCPOptions)
	total += len(backup.QoSRules)
	total += len(backup.PortGroups)
	total += len(backup.AddressSets)
	return total
}

// ListBackups lists all available backups
func (s *BackupService) ListBackups() ([]*BackupMetadata, error) {
	return s.storage.List()
}

// GetBackup retrieves backup metadata
func (s *BackupService) GetBackup(backupID string) (*BackupMetadata, error) {
	backups, err := s.storage.List()
	if err != nil {
		return nil, err
	}

	for _, backup := range backups {
		if backup.ID == backupID {
			return backup, nil
		}
	}

	return nil, fmt.Errorf("backup not found: %s", backupID)
}

// DeleteBackup removes a backup
func (s *BackupService) DeleteBackup(backupID string) error {
	return s.storage.Delete(backupID)
}

// ExportBackup exports a backup to a writer
func (s *BackupService) ExportBackup(backupID string, format BackupFormat, w io.Writer) error {
	backup, err := s.storage.Retrieve(backupID)
	if err != nil {
		return fmt.Errorf("failed to retrieve backup: %w", err)
	}

	switch format {
	case BackupFormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(backup)
	case BackupFormatYAML:
		encoder := yaml.NewEncoder(w)
		return encoder.Encode(backup)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ImportBackup imports a backup from a reader
func (s *BackupService) ImportBackup(r io.Reader, format BackupFormat) (*BackupMetadata, error) {
	var backup BackupData

	switch format {
	case BackupFormatJSON:
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(&backup); err != nil {
			return nil, fmt.Errorf("failed to decode JSON: %w", err)
		}
	case BackupFormatYAML:
		decoder := yaml.NewDecoder(r)
		if err := decoder.Decode(&backup); err != nil {
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Generate new ID for imported backup
	backup.Metadata.ID = uuid.New().String()
	backup.Metadata.CreatedAt = time.Now()

	// Store the imported backup
	options := &BackupOptions{
		Name:   backup.Metadata.Name,
		Format: format,
	}
	
	_, err := s.storage.Store(&backup, options)
	if err != nil {
		return nil, fmt.Errorf("failed to store imported backup: %w", err)
	}

	return &backup.Metadata, nil
}

// calculateChecksum calculates SHA256 checksum of backup data
func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}