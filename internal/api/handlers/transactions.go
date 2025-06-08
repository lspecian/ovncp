package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
)

type TransactionHandler struct {
	ovnService services.OVNServiceInterface
}

func NewTransactionHandler(ovnService services.OVNServiceInterface) *TransactionHandler {
	return &TransactionHandler{
		ovnService: ovnService,
	}
}

func (h *TransactionHandler) Execute(c *gin.Context) {
	var req models.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate request
	if len(req.Operations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "at least one operation is required",
		})
		return
	}

	if len(req.Operations) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "maximum 100 operations per transaction",
		})
		return
	}

	// Track operation IDs for uniqueness
	operationIDs := make(map[string]bool)

	// Validate all operations
	for i, op := range req.Operations {
		// Validate operation ID
		if op.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation failed",
				"details": fmt.Sprintf("operation %d: id is required", i),
			})
			return
		}

		if operationIDs[op.ID] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation failed",
				"details": fmt.Sprintf("operation %d: duplicate operation id '%s'", i, op.ID),
			})
			return
		}
		operationIDs[op.ID] = true

		// Validate operation type
		if !models.IsValidOperationType(op.Type) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation failed",
				"details": fmt.Sprintf("operation %s: invalid type '%s'", op.ID, op.Type),
			})
			return
		}

		// Validate resource type
		if !models.IsValidResourceType(op.Resource) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation failed",
				"details": fmt.Sprintf("operation %s: invalid resource '%s'", op.ID, op.Resource),
			})
			return
		}

		// Validate operation-specific requirements
		if err := h.validateOperation(&op); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation failed",
				"details": fmt.Sprintf("operation %s: %v", op.ID, err),
			})
			return
		}
	}

	// If dry run, return validation success
	if req.DryRun {
		c.JSON(http.StatusOK, gin.H{
			"message": "validation successful",
			"operations": len(req.Operations),
		})
		return
	}

	// Execute the transaction
	response := h.executeTransaction(c, &req)
	
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

func (h *TransactionHandler) validateOperation(op *models.TransactionOperation) error {
	switch op.Type {
	case models.OperationCreate:
		if op.Data == nil || len(op.Data) == 0 {
			return fmt.Errorf("data is required for create operation")
		}
		if op.ResourceID != "" {
			return fmt.Errorf("resource_id should not be provided for create operation")
		}
		// Validate switch_id for port/acl creation
		if op.Resource == models.ResourcePort || op.Resource == models.ResourceACL {
			if op.SwitchID == "" {
				return fmt.Errorf("switch_id is required for %s creation", op.Resource)
			}
		}

	case models.OperationUpdate:
		if op.ResourceID == "" {
			return fmt.Errorf("resource_id is required for update operation")
		}
		if op.Data == nil || len(op.Data) == 0 {
			return fmt.Errorf("data is required for update operation")
		}

	case models.OperationDelete:
		if op.ResourceID == "" {
			return fmt.Errorf("resource_id is required for delete operation")
		}
		if op.Data != nil && len(op.Data) > 0 {
			return fmt.Errorf("data should not be provided for delete operation")
		}
	}

	return nil
}

func (h *TransactionHandler) executeTransaction(c *gin.Context, req *models.TransactionRequest) *models.TransactionResponse {
	transactionID := uuid.New().String()
	response := &models.TransactionResponse{
		TransactionID: transactionID,
		Success:       true,
		Results:       make([]models.TransactionOperationResult, 0, len(req.Operations)),
		ExecutedAt:    time.Now(),
	}

	// Track created resources for potential rollback
	createdResources := make([]struct {
		Resource   string
		ResourceID string
		SwitchID   string // For ports/ACLs
	}, 0)

	// Execute each operation
	for _, op := range req.Operations {
		result := h.executeOperation(c, &op)
		response.Results = append(response.Results, result)

		if !result.Success {
			response.Success = false
			response.Error = fmt.Sprintf("operation %s failed: %s", op.ID, result.Error)
			
			// Rollback created resources
			h.rollback(c, createdResources)
			
			// Mark remaining operations as not executed
			for i := len(response.Results); i < len(req.Operations); i++ {
				response.Results = append(response.Results, models.TransactionOperationResult{
					ID:       req.Operations[i].ID,
					Type:     req.Operations[i].Type,
					Resource: req.Operations[i].Resource,
					Success:  false,
					Error:    "not executed due to previous failure",
				})
			}
			
			return response
		}

		// Track created resources
		if op.Type == models.OperationCreate && result.ResourceID != "" {
			createdResources = append(createdResources, struct {
				Resource   string
				ResourceID string
				SwitchID   string
			}{
				Resource:   op.Resource,
				ResourceID: result.ResourceID,
				SwitchID:   op.SwitchID,
			})
		}
	}

	return response
}

func (h *TransactionHandler) executeOperation(c *gin.Context, op *models.TransactionOperation) models.TransactionOperationResult {
	result := models.TransactionOperationResult{
		ID:       op.ID,
		Type:     op.Type,
		Resource: op.Resource,
		Success:  false,
	}

	switch op.Type {
	case models.OperationCreate:
		resourceID, data, err := h.createResource(c, op.Resource, op.SwitchID, op.Data)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.ResourceID = resourceID
			result.Data = data
		}

	case models.OperationUpdate:
		result.ResourceID = op.ResourceID
		data, err := h.updateResource(c, op.Resource, op.ResourceID, op.Data)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Data = data
		}

	case models.OperationDelete:
		result.ResourceID = op.ResourceID
		err := h.deleteResource(c, op.Resource, op.ResourceID)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
		}
	}

	return result
}

func (h *TransactionHandler) createResource(c *gin.Context, resource, switchID string, data map[string]interface{}) (string, map[string]interface{}, error) {
	ctx := c.Request.Context()

	switch resource {
	case models.ResourceSwitch:
		var ls models.LogicalSwitch
		if err := mapToStruct(data, &ls); err != nil {
			return "", nil, fmt.Errorf("invalid switch data: %v", err)
		}
		created, err := h.ovnService.CreateLogicalSwitch(ctx, &ls)
		if err != nil {
			return "", nil, err
		}
		return created.UUID, structToMap(created), nil

	case models.ResourceRouter:
		var lr models.LogicalRouter
		if err := mapToStruct(data, &lr); err != nil {
			return "", nil, fmt.Errorf("invalid router data: %v", err)
		}
		created, err := h.ovnService.CreateLogicalRouter(ctx, &lr)
		if err != nil {
			return "", nil, err
		}
		return created.UUID, structToMap(created), nil

	case models.ResourcePort:
		if switchID == "" {
			return "", nil, fmt.Errorf("switch_id is required for port creation")
		}
		var port models.LogicalSwitchPort
		if err := mapToStruct(data, &port); err != nil {
			return "", nil, fmt.Errorf("invalid port data: %v", err)
		}
		created, err := h.ovnService.CreatePort(ctx, switchID, &port)
		if err != nil {
			return "", nil, err
		}
		return created.UUID, structToMap(created), nil

	case models.ResourceACL:
		if switchID == "" {
			return "", nil, fmt.Errorf("switch_id is required for ACL creation")
		}
		var acl models.ACL
		if err := mapToStruct(data, &acl); err != nil {
			return "", nil, fmt.Errorf("invalid ACL data: %v", err)
		}
		created, err := h.ovnService.CreateACL(ctx, switchID, &acl)
		if err != nil {
			return "", nil, err
		}
		return created.UUID, structToMap(created), nil

	default:
		return "", nil, fmt.Errorf("unsupported resource type: %s", resource)
	}
}

func (h *TransactionHandler) updateResource(c *gin.Context, resource, resourceID string, data map[string]interface{}) (map[string]interface{}, error) {
	ctx := c.Request.Context()

	switch resource {
	case models.ResourceSwitch:
		var ls models.LogicalSwitch
		if err := mapToStruct(data, &ls); err != nil {
			return nil, fmt.Errorf("invalid switch data: %v", err)
		}
		updated, err := h.ovnService.UpdateLogicalSwitch(ctx, resourceID, &ls)
		if err != nil {
			return nil, err
		}
		return structToMap(updated), nil

	case models.ResourceRouter:
		var lr models.LogicalRouter
		if err := mapToStruct(data, &lr); err != nil {
			return nil, fmt.Errorf("invalid router data: %v", err)
		}
		updated, err := h.ovnService.UpdateLogicalRouter(ctx, resourceID, &lr)
		if err != nil {
			return nil, err
		}
		return structToMap(updated), nil

	case models.ResourcePort:
		var port models.LogicalSwitchPort
		if err := mapToStruct(data, &port); err != nil {
			return nil, fmt.Errorf("invalid port data: %v", err)
		}
		updated, err := h.ovnService.UpdatePort(ctx, resourceID, &port)
		if err != nil {
			return nil, err
		}
		return structToMap(updated), nil

	case models.ResourceACL:
		var acl models.ACL
		if err := mapToStruct(data, &acl); err != nil {
			return nil, fmt.Errorf("invalid ACL data: %v", err)
		}
		updated, err := h.ovnService.UpdateACL(ctx, resourceID, &acl)
		if err != nil {
			return nil, err
		}
		return structToMap(updated), nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resource)
	}
}

func (h *TransactionHandler) deleteResource(c *gin.Context, resource, resourceID string) error {
	ctx := c.Request.Context()

	switch resource {
	case models.ResourceSwitch:
		return h.ovnService.DeleteLogicalSwitch(ctx, resourceID)
	case models.ResourceRouter:
		return h.ovnService.DeleteLogicalRouter(ctx, resourceID)
	case models.ResourcePort:
		return h.ovnService.DeletePort(ctx, resourceID)
	case models.ResourceACL:
		return h.ovnService.DeleteACL(ctx, resourceID)
	default:
		return fmt.Errorf("unsupported resource type: %s", resource)
	}
}

func (h *TransactionHandler) rollback(c *gin.Context, createdResources []struct {
	Resource   string
	ResourceID string
	SwitchID   string
}) {
	// Delete in reverse order
	for i := len(createdResources) - 1; i >= 0; i-- {
		res := createdResources[i]
		// Best effort rollback - ignore errors
		_ = h.deleteResource(c, res.Resource, res.ResourceID)
	}
}

// handleError handles generic errors
func (h *TransactionHandler) handleError(c *gin.Context, err error) {
	// Check if client is not connected
	if strings.Contains(err.Error(), "not connected") {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "OVN service unavailable",
			"details": "unable to connect to OVN northbound database",
		})
		return
	}

	// Default error response
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal server error",
		"details": err.Error(),
	})
}

// Helper functions to convert between map and struct
func mapToStruct(m map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func structToMap(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	return m
}