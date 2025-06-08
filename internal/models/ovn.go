package models

import (
	"time"
)

type LogicalSwitch struct {
	UUID        string                 `json:"uuid"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	VLAN        int                    `json:"vlan,omitempty"`
	Ports       []string               `json:"ports,omitempty"`
	ACLs        []string               `json:"acls,omitempty"`
	QoSRules    []string               `json:"qos_rules,omitempty"`
	LoadBalancer []string              `json:"load_balancer,omitempty"`
	DNSRecords  []string               `json:"dns_records,omitempty"`
	OtherConfig map[string]string      `json:"other_config,omitempty"`
	ExternalIDs map[string]string      `json:"external_ids,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type LogicalRouter struct {
	UUID          string                 `json:"uuid"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Ports         []string               `json:"ports,omitempty"`
	StaticRoutes  []StaticRoute          `json:"static_routes,omitempty"`
	Policies      []string               `json:"policies,omitempty"`
	NAT           []NAT                  `json:"nat,omitempty"`
	LoadBalancer  []string               `json:"load_balancer,omitempty"`
	Options       map[string]string      `json:"options,omitempty"`
	ExternalIDs   map[string]string      `json:"external_ids,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type LogicalSwitchPort struct {
	UUID             string                 `json:"uuid"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	MAC              string                 `json:"mac,omitempty"`
	SwitchID         string                 `json:"switch_id,omitempty"`
	Addresses        []string               `json:"addresses"`
	PortSecurity     []string               `json:"port_security,omitempty"`
	Up               *bool                  `json:"up,omitempty"`
	Enabled          *bool                  `json:"enabled,omitempty"`
	DHCPv4Options    *string                `json:"dhcpv4_options,omitempty"`
	DHCPv6Options    *string                `json:"dhcpv6_options,omitempty"`
	Options          map[string]string      `json:"options,omitempty"`
	ExternalIDs      map[string]string      `json:"external_ids,omitempty"`
	ParentName       string                 `json:"parent_name,omitempty"`
	Tag              int                    `json:"tag,omitempty"`
	ParentUUID       string                 `json:"parent_uuid,omitempty"` // For compatibility with cached service
	ParentType       string                 `json:"parent_type,omitempty"` // For compatibility with cached service
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type LogicalRouterPort struct {
	UUID        string                 `json:"uuid"`
	Name        string                 `json:"name"`
	MAC         string                 `json:"mac"`
	Networks    []string               `json:"networks"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	PeerPort    string                 `json:"peer,omitempty"`
	Options     map[string]string      `json:"options,omitempty"`
	ExternalIDs map[string]string      `json:"external_ids,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ACL struct {
	UUID        string                 `json:"uuid"`
	Name        string                 `json:"name,omitempty"`
	Priority    int                    `json:"priority"`
	Direction   string                 `json:"direction"`
	Match       string                 `json:"match"`
	Action      string                 `json:"action"`
	Log         bool                   `json:"log"`
	Severity    string                 `json:"severity,omitempty"`
	Alert       bool                   `json:"alert"`
	ExternalIDs map[string]string      `json:"external_ids,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type StaticRoute struct {
	IPPrefix   string                 `json:"ip_prefix"`
	Nexthop    string                 `json:"nexthop"`
	OutputPort *string                `json:"output_port,omitempty"`
	Policy     *string                `json:"policy,omitempty"`
}

type NAT struct {
	UUID         string                 `json:"uuid"`
	Type         string                 `json:"type"`
	ExternalIP   string                 `json:"external_ip"`
	ExternalMAC  *string                `json:"external_mac,omitempty"`
	LogicalIP    string                 `json:"logical_ip"`
	LogicalPort  *string                `json:"logical_port,omitempty"`
	ExternalIDs  map[string]string      `json:"external_ids,omitempty"`
}

type LoadBalancer struct {
	UUID         string                 `json:"uuid"`
	Name         string                 `json:"name"`
	VIPs         map[string]string      `json:"vips"`
	Protocol     *string                `json:"protocol,omitempty"`
	HealthCheck  []HealthCheck          `json:"health_check,omitempty"`
	IPPortMappings map[string]string    `json:"ip_port_mappings,omitempty"`
	SelectionFields []string             `json:"selection_fields,omitempty"`
	Options      map[string]string      `json:"options,omitempty"`
	ExternalIDs  map[string]string      `json:"external_ids,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type HealthCheck struct {
	VIP      string                 `json:"vip"`
	Options  map[string]string      `json:"options,omitempty"`
}

type DHCPOptions struct {
	UUID        string                 `json:"uuid"`
	CIDR        string                 `json:"cidr"`
	Options     map[string]string      `json:"options"`
	ExternalIDs map[string]string      `json:"external_ids,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type QoSRule struct {
	UUID        string                 `json:"uuid"`
	Priority    int                    `json:"priority"`
	Direction   string                 `json:"direction"`
	Match       string                 `json:"match"`
	Action      map[string]interface{} `json:"action"`
	Bandwidth   map[string]int         `json:"bandwidth,omitempty"`
	ExternalIDs map[string]string      `json:"external_ids,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}