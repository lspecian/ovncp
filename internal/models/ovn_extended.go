package models

import "time"

// QoS represents QoS rules
type QoS struct {
	UUID        string            `json:"uuid"`
	Priority    int               `json:"priority"`
	Direction   string            `json:"direction"` // from-lport, to-lport
	Match       string            `json:"match"`
	Action      map[string]string `json:"action"`
	Bandwidth   map[string]int    `json:"bandwidth,omitempty"`
	ExternalIDs map[string]string `json:"external_ids,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PortGroup represents a group of ports
type PortGroup struct {
	UUID        string            `json:"uuid"`
	Name        string            `json:"name"`
	Ports       []string          `json:"ports"`
	ACLs        []string          `json:"acls,omitempty"`
	ExternalIDs map[string]string `json:"external_ids,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// AddressSet represents a set of IP addresses
type AddressSet struct {
	UUID        string            `json:"uuid"`
	Name        string            `json:"name"`
	Addresses   []string          `json:"addresses"`
	ExternalIDs map[string]string `json:"external_ids,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}