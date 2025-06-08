package templates

import (
	"encoding/json"
	"fmt"
)

// PolicyTemplate represents a reusable network policy template
type PolicyTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags"`
	Variables   []TemplateVariable     `json:"variables"`
	Rules       []TemplateRule         `json:"rules"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TemplateVariable represents a variable that can be customized
type TemplateVariable struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"` // string, number, ipv4, ipv6, cidr, port, mac
	Required     bool        `json:"required"`
	Default      interface{} `json:"default,omitempty"`
	Validation   string      `json:"validation,omitempty"`
	Example      string      `json:"example,omitempty"`
}

// TemplateRule represents a rule template
type TemplateRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Direction   string `json:"direction"` // ingress, egress
	Action      string `json:"action"`    // allow, drop, reject
	Match       string `json:"match"`     // OVN match expression with variables
	Log         bool   `json:"log"`
}

// PolicyTemplateLibrary provides access to pre-defined templates
type PolicyTemplateLibrary struct {
	templates map[string]*PolicyTemplate
}

// NewPolicyTemplateLibrary creates a new template library
func NewPolicyTemplateLibrary() *PolicyTemplateLibrary {
	lib := &PolicyTemplateLibrary{
		templates: make(map[string]*PolicyTemplate),
	}
	
	// Load built-in templates
	lib.loadBuiltinTemplates()
	
	return lib
}

// loadBuiltinTemplates loads all built-in policy templates
func (l *PolicyTemplateLibrary) loadBuiltinTemplates() {
	// Web Server Template
	l.templates["web-server"] = &PolicyTemplate{
		ID:          "web-server",
		Name:        "Web Server",
		Description: "Standard security policy for web servers (HTTP/HTTPS)",
		Category:    "Application",
		Tags:        []string{"web", "http", "https", "server"},
		Variables: []TemplateVariable{
			{
				Name:        "server_ip",
				Description: "IP address of the web server",
				Type:        "ipv4",
				Required:    true,
				Example:     "10.0.1.10",
			},
			{
				Name:        "allowed_sources",
				Description: "CIDR blocks allowed to access the web server",
				Type:        "cidr",
				Required:    false,
				Default:     "0.0.0.0/0",
				Example:     "192.168.0.0/16",
			},
			{
				Name:        "enable_ssh",
				Description: "Allow SSH access for management",
				Type:        "boolean",
				Required:    false,
				Default:     false,
			},
			{
				Name:        "ssh_sources",
				Description: "CIDR blocks allowed for SSH access",
				Type:        "cidr",
				Required:    false,
				Default:     "10.0.0.0/8",
				Example:     "10.0.100.0/24",
			},
		},
		Rules: []TemplateRule{
			{
				Name:        "allow-http",
				Description: "Allow HTTP traffic",
				Priority:    2000,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{server_ip}} && tcp.dst == 80 && ip4.src == {{allowed_sources}}",
			},
			{
				Name:        "allow-https",
				Description: "Allow HTTPS traffic",
				Priority:    2000,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{server_ip}} && tcp.dst == 443 && ip4.src == {{allowed_sources}}",
			},
			{
				Name:        "allow-ssh",
				Description: "Allow SSH for management",
				Priority:    1900,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "{{if enable_ssh}}ip4.dst == {{server_ip}} && tcp.dst == 22 && ip4.src == {{ssh_sources}}{{else}}0{{end}}",
			},
			{
				Name:        "allow-established",
				Description: "Allow established connections",
				Priority:    1800,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ct.est && ct.rpl && ip4.dst == {{server_ip}}",
			},
			{
				Name:        "allow-icmp",
				Description: "Allow ICMP for diagnostics",
				Priority:    1700,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{server_ip}} && icmp4",
			},
			{
				Name:        "default-deny",
				Description: "Deny all other traffic",
				Priority:    100,
				Direction:   "ingress",
				Action:      "drop",
				Match:       "ip4.dst == {{server_ip}}",
				Log:         true,
			},
		},
	}

	// Database Server Template
	l.templates["database-server"] = &PolicyTemplate{
		ID:          "database-server",
		Name:        "Database Server",
		Description: "Security policy for database servers (MySQL/PostgreSQL)",
		Category:    "Application",
		Tags:        []string{"database", "mysql", "postgresql", "mariadb"},
		Variables: []TemplateVariable{
			{
				Name:        "db_ip",
				Description: "IP address of the database server",
				Type:        "ipv4",
				Required:    true,
				Example:     "10.0.2.10",
			},
			{
				Name:        "db_port",
				Description: "Database port (3306 for MySQL, 5432 for PostgreSQL)",
				Type:        "port",
				Required:    true,
				Default:     3306,
				Example:     "3306",
			},
			{
				Name:        "app_subnet",
				Description: "Application server subnet allowed to connect",
				Type:        "cidr",
				Required:    true,
				Example:     "10.0.1.0/24",
			},
			{
				Name:        "backup_server",
				Description: "Backup server IP address",
				Type:        "ipv4",
				Required:    false,
				Example:     "10.0.100.50",
			},
			{
				Name:        "enable_replication",
				Description: "Enable database replication",
				Type:        "boolean",
				Required:    false,
				Default:     false,
			},
			{
				Name:        "replica_ips",
				Description: "Comma-separated list of replica IPs",
				Type:        "string",
				Required:    false,
				Example:     "10.0.2.11,10.0.2.12",
			},
		},
		Rules: []TemplateRule{
			{
				Name:        "allow-db-from-app",
				Description: "Allow database connections from application servers",
				Priority:    2000,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{db_ip}} && tcp.dst == {{db_port}} && ip4.src == {{app_subnet}}",
			},
			{
				Name:        "allow-backup",
				Description: "Allow backup server connections",
				Priority:    1900,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "{{if backup_server}}ip4.dst == {{db_ip}} && tcp.dst == {{db_port}} && ip4.src == {{backup_server}}{{else}}0{{end}}",
			},
			{
				Name:        "allow-replication",
				Description: "Allow database replication",
				Priority:    1900,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "{{if enable_replication}}ip4.dst == {{db_ip}} && tcp.dst == {{db_port}} && ip4.src == {{{replica_ips}}}{{else}}0{{end}}",
			},
			{
				Name:        "allow-established",
				Description: "Allow established connections",
				Priority:    1800,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ct.est && ct.rpl && ip4.dst == {{db_ip}}",
			},
			{
				Name:        "deny-all",
				Description: "Deny all other database access",
				Priority:    100,
				Direction:   "ingress",
				Action:      "drop",
				Match:       "ip4.dst == {{db_ip}} && tcp.dst == {{db_port}}",
				Log:         true,
			},
		},
	}

	// Microservices Template
	l.templates["microservice"] = &PolicyTemplate{
		ID:          "microservice",
		Name:        "Microservice",
		Description: "Security policy for microservices with service mesh",
		Category:    "Application",
		Tags:        []string{"microservice", "api", "rest", "grpc"},
		Variables: []TemplateVariable{
			{
				Name:        "service_name",
				Description: "Name of the microservice",
				Type:        "string",
				Required:    true,
				Example:     "user-service",
			},
			{
				Name:        "service_ip",
				Description: "IP address of the service",
				Type:        "ipv4",
				Required:    true,
				Example:     "10.0.3.10",
			},
			{
				Name:        "service_port",
				Description: "Service port",
				Type:        "port",
				Required:    true,
				Default:     8080,
			},
			{
				Name:        "health_port",
				Description: "Health check port",
				Type:        "port",
				Required:    false,
				Default:     8081,
			},
			{
				Name:        "allowed_services",
				Description: "Comma-separated list of allowed source service IPs",
				Type:        "string",
				Required:    true,
				Example:     "10.0.3.11,10.0.3.12,10.0.3.13",
			},
			{
				Name:        "metrics_port",
				Description: "Metrics/Prometheus port",
				Type:        "port",
				Required:    false,
				Default:     9090,
			},
			{
				Name:        "monitoring_subnet",
				Description: "Monitoring system subnet",
				Type:        "cidr",
				Required:    false,
				Default:     "10.0.200.0/24",
			},
		},
		Rules: []TemplateRule{
			{
				Name:        "allow-service-mesh",
				Description: "Allow traffic from authorized services",
				Priority:    2000,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{service_ip}} && tcp.dst == {{service_port}} && ip4.src == {{{allowed_services}}}",
			},
			{
				Name:        "allow-health-checks",
				Description: "Allow health check probes",
				Priority:    1900,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{service_ip}} && tcp.dst == {{health_port}}",
			},
			{
				Name:        "allow-metrics",
				Description: "Allow metrics collection",
				Priority:    1800,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{service_ip}} && tcp.dst == {{metrics_port}} && ip4.src == {{monitoring_subnet}}",
			},
			{
				Name:        "allow-dns",
				Description: "Allow DNS lookups",
				Priority:    1700,
				Direction:   "egress",
				Action:      "allow",
				Match:       "ip4.src == {{service_ip}} && udp.dst == 53",
			},
			{
				Name:        "allow-service-egress",
				Description: "Allow outbound connections to other services",
				Priority:    1600,
				Direction:   "egress",
				Action:      "allow",
				Match:       "ip4.src == {{service_ip}} && ip4.dst == {{{allowed_services}}}",
			},
			{
				Name:        "deny-all-ingress",
				Description: "Deny all other inbound traffic",
				Priority:    100,
				Direction:   "ingress",
				Action:      "drop",
				Match:       "ip4.dst == {{service_ip}}",
			},
		},
	}

	// DMZ Template
	l.templates["dmz"] = &PolicyTemplate{
		ID:          "dmz",
		Name:        "DMZ Zone",
		Description: "Security policy for DMZ (Demilitarized Zone) networks",
		Category:    "Network Zone",
		Tags:        []string{"dmz", "perimeter", "security", "zone"},
		Variables: []TemplateVariable{
			{
				Name:        "dmz_subnet",
				Description: "DMZ subnet CIDR",
				Type:        "cidr",
				Required:    true,
				Example:     "172.16.0.0/24",
			},
			{
				Name:        "internal_subnets",
				Description: "Internal network subnets (comma-separated)",
				Type:        "string",
				Required:    true,
				Example:     "10.0.0.0/16,192.168.0.0/16",
			},
			{
				Name:        "allowed_dmz_to_internal_ports",
				Description: "Ports DMZ can access in internal network",
				Type:        "string",
				Required:    false,
				Default:     "443,636,389",
			},
		},
		Rules: []TemplateRule{
			{
				Name:        "allow-internet-to-dmz",
				Description: "Allow internet traffic to DMZ services",
				Priority:    2000,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{dmz_subnet}} && (tcp.dst == 80 || tcp.dst == 443)",
			},
			{
				Name:        "deny-dmz-to-internal",
				Description: "Block DMZ to internal by default",
				Priority:    1900,
				Direction:   "egress",
				Action:      "drop",
				Match:       "ip4.src == {{dmz_subnet}} && ip4.dst == {{{internal_subnets}}}",
				Log:         true,
			},
			{
				Name:        "allow-dmz-to-internal-limited",
				Description: "Allow specific DMZ to internal traffic",
				Priority:    2000,
				Direction:   "egress",
				Action:      "allow",
				Match:       "ip4.src == {{dmz_subnet}} && ip4.dst == {{{internal_subnets}}} && tcp.dst == {{{allowed_dmz_to_internal_ports}}}",
			},
			{
				Name:        "allow-dmz-to-internet",
				Description: "Allow DMZ to internet",
				Priority:    1800,
				Direction:   "egress",
				Action:      "allow",
				Match:       "ip4.src == {{dmz_subnet}} && !(ip4.dst == {{{internal_subnets}}})",
			},
		},
	}

	// Zero Trust Template
	l.templates["zero-trust"] = &PolicyTemplate{
		ID:          "zero-trust",
		Name:        "Zero Trust Network",
		Description: "Zero trust security model - deny by default, explicit allow",
		Category:    "Security Model",
		Tags:        []string{"zero-trust", "security", "strict"},
		Variables: []TemplateVariable{
			{
				Name:        "resource_ip",
				Description: "IP of the protected resource",
				Type:        "ipv4",
				Required:    true,
			},
			{
				Name:        "resource_port",
				Description: "Port of the protected resource",
				Type:        "port",
				Required:    true,
			},
			{
				Name:        "authorized_users",
				Description: "Authorized user IPs (comma-separated)",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "require_encryption",
				Description: "Require encrypted connections only",
				Type:        "boolean",
				Required:    false,
				Default:     true,
			},
		},
		Rules: []TemplateRule{
			{
				Name:        "deny-all-by-default",
				Description: "Deny all traffic by default",
				Priority:    100,
				Direction:   "ingress",
				Action:      "drop",
				Match:       "ip4.dst == {{resource_ip}}",
				Log:         true,
			},
			{
				Name:        "allow-authorized-users",
				Description: "Allow only authorized users",
				Priority:    2000,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.dst == {{resource_ip}} && tcp.dst == {{resource_port}} && ip4.src == {{{authorized_users}}}",
				Log:         true,
			},
			{
				Name:        "verify-encryption",
				Description: "Ensure connections are encrypted",
				Priority:    1900,
				Direction:   "ingress",
				Action:      "drop",
				Match:       "{{if require_encryption}}ip4.dst == {{resource_ip}} && tcp.dst != 443 && tcp.dst != 22{{else}}0{{end}}",
			},
		},
	}

	// Kubernetes Pod Network Policy
	l.templates["k8s-pod"] = &PolicyTemplate{
		ID:          "k8s-pod",
		Name:        "Kubernetes Pod",
		Description: "Network policy for Kubernetes pods",
		Category:    "Container",
		Tags:        []string{"kubernetes", "k8s", "pod", "container"},
		Variables: []TemplateVariable{
			{
				Name:        "pod_cidr",
				Description: "Pod network CIDR",
				Type:        "cidr",
				Required:    true,
				Example:     "10.244.0.0/16",
			},
			{
				Name:        "service_cidr",
				Description: "Service network CIDR",
				Type:        "cidr",
				Required:    true,
				Example:     "10.96.0.0/12",
			},
			{
				Name:        "namespace",
				Description: "Kubernetes namespace",
				Type:        "string",
				Required:    true,
				Example:     "production",
			},
			{
				Name:        "pod_selector",
				Description: "Pod label selector",
				Type:        "string",
				Required:    true,
				Example:     "app=nginx",
			},
		},
		Rules: []TemplateRule{
			{
				Name:        "allow-dns",
				Description: "Allow DNS resolution",
				Priority:    2000,
				Direction:   "egress",
				Action:      "allow",
				Match:       "ip4.src == {{pod_cidr}} && udp.dst == 53",
			},
			{
				Name:        "allow-same-namespace",
				Description: "Allow traffic within same namespace",
				Priority:    1900,
				Direction:   "ingress",
				Action:      "allow",
				Match:       "ip4.src == {{pod_cidr}} && ip4.dst == {{pod_cidr}}",
			},
			{
				Name:        "allow-services",
				Description: "Allow access to Kubernetes services",
				Priority:    1800,
				Direction:   "egress",
				Action:      "allow",
				Match:       "ip4.src == {{pod_cidr}} && ip4.dst == {{service_cidr}}",
			},
			{
				Name:        "deny-other-namespaces",
				Description: "Deny traffic from other namespaces",
				Priority:    1000,
				Direction:   "ingress",
				Action:      "drop",
				Match:       "ip4.dst == {{pod_cidr}}",
			},
		},
	}
}

// GetTemplate returns a template by ID
func (l *PolicyTemplateLibrary) GetTemplate(id string) (*PolicyTemplate, error) {
	template, exists := l.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return template, nil
}

// ListTemplates returns all available templates
func (l *PolicyTemplateLibrary) ListTemplates() []*PolicyTemplate {
	templates := make([]*PolicyTemplate, 0, len(l.templates))
	for _, template := range l.templates {
		templates = append(templates, template)
	}
	return templates
}

// ListTemplatesByCategory returns templates in a specific category
func (l *PolicyTemplateLibrary) ListTemplatesByCategory(category string) []*PolicyTemplate {
	templates := []*PolicyTemplate{}
	for _, template := range l.templates {
		if template.Category == category {
			templates = append(templates, template)
		}
	}
	return templates
}

// SearchTemplates searches templates by tags
func (l *PolicyTemplateLibrary) SearchTemplates(tags []string) []*PolicyTemplate {
	templates := []*PolicyTemplate{}
	for _, template := range l.templates {
		for _, tag := range tags {
			for _, templateTag := range template.Tags {
				if tag == templateTag {
					templates = append(templates, template)
					break
				}
			}
		}
	}
	return templates
}

// AddCustomTemplate adds a custom template to the library
func (l *PolicyTemplateLibrary) AddCustomTemplate(template *PolicyTemplate) error {
	if _, exists := l.templates[template.ID]; exists {
		return fmt.Errorf("template with ID %s already exists", template.ID)
	}
	l.templates[template.ID] = template
	return nil
}

// ExportTemplate exports a template as JSON
func (l *PolicyTemplateLibrary) ExportTemplate(id string) ([]byte, error) {
	template, err := l.GetTemplate(id)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(template, "", "  ")
}

// ImportTemplate imports a template from JSON
func (l *PolicyTemplateLibrary) ImportTemplate(data []byte) (*PolicyTemplate, error) {
	var template PolicyTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	
	if err := l.AddCustomTemplate(&template); err != nil {
		return nil, err
	}
	
	return &template, nil
}