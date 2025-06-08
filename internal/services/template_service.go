package services

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/templates"
	"go.uber.org/zap"
)

// TemplateService handles policy template operations
type TemplateService struct {
	library    *templates.PolicyTemplateLibrary
	ovnService OVNServiceInterface
	logger     *zap.Logger
}

// NewTemplateService creates a new template service
func NewTemplateService(ovnService OVNServiceInterface, logger *zap.Logger) *TemplateService {
	return &TemplateService{
		library:    templates.NewPolicyTemplateLibrary(),
		ovnService: ovnService,
		logger:     logger,
	}
}

// TemplateInstance represents an instantiated template with variables filled
type TemplateInstance struct {
	TemplateID  string                 `json:"template_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Variables   map[string]interface{} `json:"variables"`
	Rules       []*models.ACL          `json:"rules"`
}

// TemplateValidationResult represents validation results
type TemplateValidationResult struct {
	Valid    bool                     `json:"valid"`
	Errors   map[string]string        `json:"errors,omitempty"`
	Warnings []string                 `json:"warnings,omitempty"`
	Preview  []*models.ACL            `json:"preview,omitempty"`
}

// ListTemplates returns all available templates
func (s *TemplateService) ListTemplates() []*templates.PolicyTemplate {
	return s.library.ListTemplates()
}

// GetTemplate returns a specific template by ID
func (s *TemplateService) GetTemplate(id string) (*templates.PolicyTemplate, error) {
	return s.library.GetTemplate(id)
}

// ListTemplatesByCategory returns templates in a specific category
func (s *TemplateService) ListTemplatesByCategory(category string) []*templates.PolicyTemplate {
	return s.library.ListTemplatesByCategory(category)
}

// SearchTemplates searches templates by tags
func (s *TemplateService) SearchTemplates(tags []string) []*templates.PolicyTemplate {
	return s.library.SearchTemplates(tags)
}

// ValidateTemplate validates template variables
func (s *TemplateService) ValidateTemplate(templateID string, variables map[string]interface{}) (*TemplateValidationResult, error) {
	template, err := s.library.GetTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	result := &TemplateValidationResult{
		Valid:  true,
		Errors: make(map[string]string),
	}

	// Validate all required variables are provided
	for _, v := range template.Variables {
		value, exists := variables[v.Name]
		if v.Required && !exists {
			result.Valid = false
			result.Errors[v.Name] = "required variable not provided"
			continue
		}

		if exists {
			// Validate variable type
			if err := s.validateVariable(v, value); err != nil {
				result.Valid = false
				result.Errors[v.Name] = err.Error()
			}
		} else if v.Default != nil {
			// Use default value
			variables[v.Name] = v.Default
		}
	}

	// Generate preview if validation passes
	if result.Valid {
		preview, err := s.generateRules(template, variables)
		if err != nil {
			result.Valid = false
			result.Errors["_generation"] = err.Error()
		} else {
			result.Preview = preview
		}
	}

	return result, nil
}

// InstantiateTemplate creates ACL rules from a template
func (s *TemplateService) InstantiateTemplate(ctx context.Context, templateID string, variables map[string]interface{}, targetSwitch string) (*TemplateInstance, error) {
	template, err := s.library.GetTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Validate template
	validation, err := s.ValidateTemplate(templateID, variables)
	if err != nil {
		return nil, err
	}

	if !validation.Valid {
		return nil, fmt.Errorf("template validation failed: %v", validation.Errors)
	}

	// Generate rules
	rules, err := s.generateRules(template, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rules: %w", err)
	}

	// Apply rules to switch if specified
	if targetSwitch != "" {
		for _, rule := range rules {
			_, err := s.ovnService.CreateACL(ctx, targetSwitch, rule)
			if err != nil {
				s.logger.Error("Failed to create ACL",
					zap.String("rule", rule.Name),
					zap.Error(err))
				// Continue with other rules
			}
		}
	}

	instance := &TemplateInstance{
		TemplateID:  template.ID,
		Name:        template.Name,
		Description: template.Description,
		Variables:   variables,
		Rules:       rules,
	}

	return instance, nil
}

// generateRules generates ACL rules from template
func (s *TemplateService) generateRules(template *templates.PolicyTemplate, variables map[string]interface{}) ([]*models.ACL, error) {
	var rules []*models.ACL

	for _, templateRule := range template.Rules {
		// Process match expression
		match, err := s.processTemplate(templateRule.Match, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to process match for rule %s: %w", templateRule.Name, err)
		}

		// Skip rules with "0" match (disabled)
		if match == "0" {
			continue
		}

		acl := &models.ACL{
			Name:        templateRule.Name,
			Direction:   templateRule.Direction,
			Priority:    templateRule.Priority,
			Match:       match,
			Action:      templateRule.Action,
			Log:         templateRule.Log,
			ExternalIDs: map[string]string{
				"template":    template.ID,
				"description": templateRule.Description,
			},
		}

		rules = append(rules, acl)
	}

	return rules, nil
}

// processTemplate processes template strings with variables
func (s *TemplateService) processTemplate(templateStr string, variables map[string]interface{}) (string, error) {
	// Create custom template functions
	funcMap := template.FuncMap{
		"join": strings.Join,
		"split": strings.Split,
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
	}

	// Create template
	tmpl, err := template.New("rule").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("invalid template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	// Process result for OVN syntax
	result := buf.String()
	
	// Handle comma-separated lists in triple braces
	re := regexp.MustCompile(`\{\{\{([^}]+)\}\}\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		// Extract the variable name
		varName := strings.Trim(match, "{}")
		if val, ok := variables[varName]; ok {
			// Convert to OVN list format
			if strVal, ok := val.(string); ok {
				// Split by comma and format for OVN
				items := strings.Split(strVal, ",")
				var formatted []string
				for _, item := range items {
					formatted = append(formatted, strings.TrimSpace(item))
				}
				return "{" + strings.Join(formatted, ", ") + "}"
			}
		}
		return match
	})

	return result, nil
}

// validateVariable validates a single variable
func (s *TemplateService) validateVariable(varDef templates.TemplateVariable, value interface{}) error {
	switch varDef.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}

	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}

	case "ipv4":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for IPv4, got %T", value)
		}
		if !isValidIPv4(str) {
			return fmt.Errorf("invalid IPv4 address: %s", str)
		}

	case "ipv6":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for IPv6, got %T", value)
		}
		if !isValidIPv6(str) {
			return fmt.Errorf("invalid IPv6 address: %s", str)
		}

	case "cidr":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for CIDR, got %T", value)
		}
		if !isValidCIDR(str) {
			return fmt.Errorf("invalid CIDR: %s", str)
		}

	case "port":
		var port int
		switch v := value.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		case string:
			fmt.Sscanf(v, "%d", &port)
		default:
			return fmt.Errorf("expected number for port, got %T", value)
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535, got %d", port)
		}

	case "mac":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for MAC, got %T", value)
		}
		if !isValidMAC(str) {
			return fmt.Errorf("invalid MAC address: %s", str)
		}
	}

	// Run custom validation if specified
	if varDef.Validation != "" {
		// This could be extended to support regex or custom validation functions
		re, err := regexp.Compile(varDef.Validation)
		if err == nil {
			if str, ok := value.(string); ok {
				if !re.MatchString(str) {
					return fmt.Errorf("value does not match validation pattern: %s", varDef.Validation)
				}
			}
		}
	}

	return nil
}

// Helper functions for validation
func isValidIPv4(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err != nil {
			return false
		}
		if num < 0 || num > 255 {
			return false
		}
	}
	return true
}

func isValidIPv6(ip string) bool {
	// Simplified IPv6 validation
	return strings.Contains(ip, ":") && len(ip) >= 3
}

func isValidCIDR(cidr string) bool {
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return false
	}
	if !isValidIPv4(parts[0]) && !isValidIPv6(parts[0]) {
		return false
	}
	var prefix int
	if _, err := fmt.Sscanf(parts[1], "%d", &prefix); err != nil {
		return false
	}
	if isValidIPv4(parts[0]) && (prefix < 0 || prefix > 32) {
		return false
	}
	if isValidIPv6(parts[0]) && (prefix < 0 || prefix > 128) {
		return false
	}
	return true
}

func isValidMAC(mac string) bool {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return false
	}
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
		var num int
		if _, err := fmt.Sscanf(part, "%x", &num); err != nil {
			return false
		}
	}
	return true
}

// ImportTemplate imports a custom template
func (s *TemplateService) ImportTemplate(data []byte) (*templates.PolicyTemplate, error) {
	return s.library.ImportTemplate(data)
}

// ExportTemplate exports a template as JSON
func (s *TemplateService) ExportTemplate(id string) ([]byte, error) {
	return s.library.ExportTemplate(id)
}