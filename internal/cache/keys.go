package cache

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Cache key prefixes
const (
	PrefixSwitch       = "switch:"
	PrefixRouter       = "router:"
	PrefixPort         = "port:"
	PrefixACL          = "acl:"
	PrefixLoadBalancer = "lb:"
	PrefixNAT          = "nat:"
	PrefixUser         = "user:"
	PrefixSession      = "session:"
	PrefixTopology     = "topology:"
	PrefixMetrics      = "metrics:"
)

// Cache TTLs
const (
	TTLShort     = 30 * time.Second  // For frequently changing data
	TTLMedium    = 5 * time.Minute   // For moderately stable data
	TTLLong      = 30 * time.Minute  // For stable data
	TTLSession   = 24 * time.Hour    // For user sessions
	TTLPermanent = 0                 // No expiration
)

// Key builders

// SwitchKey returns cache key for a logical switch
func SwitchKey(uuid string) string {
	return PrefixSwitch + uuid
}

// SwitchListKey returns cache key for switch list
func SwitchListKey(page, pageSize int, filters map[string]string) string {
	parts := []string{PrefixSwitch, "list", fmt.Sprintf("p%d_s%d", page, pageSize)}
	
	// Add sorted filters to ensure consistent keys
	for k, v := range filters {
		parts = append(parts, fmt.Sprintf("%s:%s", k, v))
	}
	
	return strings.Join(parts, ":")
}

// RouterKey returns cache key for a logical router
func RouterKey(uuid string) string {
	return PrefixRouter + uuid
}

// RouterListKey returns cache key for router list
func RouterListKey(page, pageSize int, filters map[string]string) string {
	parts := []string{PrefixRouter, "list", fmt.Sprintf("p%d_s%d", page, pageSize)}
	
	for k, v := range filters {
		parts = append(parts, fmt.Sprintf("%s:%s", k, v))
	}
	
	return strings.Join(parts, ":")
}

// PortKey returns cache key for a port
func PortKey(uuid string) string {
	return PrefixPort + uuid
}

// PortListKey returns cache key for port list
func PortListKey(parentUUID string, parentType string) string {
	return fmt.Sprintf("%slist:%s:%s", PrefixPort, parentType, parentUUID)
}

// ACLKey returns cache key for an ACL
func ACLKey(uuid string) string {
	return PrefixACL + uuid
}

// ACLListKey returns cache key for ACL list
func ACLListKey(filters map[string]string) string {
	parts := []string{PrefixACL, "list"}
	
	for k, v := range filters {
		parts = append(parts, fmt.Sprintf("%s:%s", k, v))
	}
	
	return strings.Join(parts, ":")
}

// LoadBalancerKey returns cache key for a load balancer
func LoadBalancerKey(uuid string) string {
	return PrefixLoadBalancer + uuid
}

// LoadBalancerListKey returns cache key for load balancer list
func LoadBalancerListKey() string {
	return PrefixLoadBalancer + "list"
}

// NATKey returns cache key for a NAT rule
func NATKey(uuid string) string {
	return PrefixNAT + uuid
}

// NATListKey returns cache key for NAT rules list
func NATListKey(routerUUID string) string {
	return fmt.Sprintf("%slist:%s", PrefixNAT, routerUUID)
}

// UserKey returns cache key for user data
func UserKey(userID string) string {
	return PrefixUser + userID
}

// SessionKey returns cache key for user session
func SessionKey(sessionID string) string {
	return PrefixSession + sessionID
}

// TopologyKey returns cache key for network topology
func TopologyKey() string {
	return PrefixTopology + "full"
}

// MetricsKey returns cache key for metrics data
func MetricsKey(metricType string, resourceID string) string {
	return fmt.Sprintf("%s%s:%s", PrefixMetrics, metricType, resourceID)
}

// Pattern builders for cache invalidation

// SwitchPattern returns pattern to match all switch-related keys
func SwitchPattern() string {
	return PrefixSwitch + "*"
}

// RouterPattern returns pattern to match all router-related keys
func RouterPattern() string {
	return PrefixRouter + "*"
}

// PortPattern returns pattern to match all port-related keys
func PortPattern() string {
	return PrefixPort + "*"
}

// PortsByParentPattern returns pattern to match ports of a specific parent
func PortsByParentPattern(parentUUID string) string {
	return fmt.Sprintf("%slist:*:%s", PrefixPort, parentUUID)
}

// ACLPattern returns pattern to match all ACL-related keys
func ACLPattern() string {
	return PrefixACL + "*"
}

// TopologyPattern returns pattern to match topology keys
func TopologyPattern() string {
	return PrefixTopology + "*"
}

// CacheKeyInfo represents information about a cache key
type CacheKeyInfo struct {
	Key         string
	TTL         time.Duration
	Invalidates []string // Patterns to invalidate when this key changes
}

// GetCacheKeyInfo returns caching information for different resource types
func GetCacheKeyInfo(resourceType string, operation string) CacheKeyInfo {
	switch resourceType {
	case "switch":
		switch operation {
		case "create", "update", "delete":
			return CacheKeyInfo{
				TTL: 0, // Don't cache write operations
				Invalidates: []string{
					SwitchPattern(),
					TopologyPattern(),
				},
			}
		case "get":
			return CacheKeyInfo{
				TTL:         TTLMedium,
				Invalidates: []string{},
			}
		case "list":
			return CacheKeyInfo{
				TTL:         TTLShort,
				Invalidates: []string{},
			}
		}
		
	case "router":
		switch operation {
		case "create", "update", "delete":
			return CacheKeyInfo{
				TTL: 0,
				Invalidates: []string{
					RouterPattern(),
					TopologyPattern(),
				},
			}
		case "get":
			return CacheKeyInfo{
				TTL:         TTLMedium,
				Invalidates: []string{},
			}
		case "list":
			return CacheKeyInfo{
				TTL:         TTLShort,
				Invalidates: []string{},
			}
		}
		
	case "port":
		switch operation {
		case "create", "update", "delete":
			return CacheKeyInfo{
				TTL: 0,
				Invalidates: []string{
					PortPattern(),
					TopologyPattern(),
				},
			}
		case "get":
			return CacheKeyInfo{
				TTL:         TTLMedium,
				Invalidates: []string{},
			}
		case "list":
			return CacheKeyInfo{
				TTL:         TTLShort,
				Invalidates: []string{},
			}
		}
		
	case "acl":
		switch operation {
		case "create", "update", "delete":
			return CacheKeyInfo{
				TTL: 0,
				Invalidates: []string{
					ACLPattern(),
				},
			}
		case "get":
			return CacheKeyInfo{
				TTL:         TTLLong, // ACLs change less frequently
				Invalidates: []string{},
			}
		case "list":
			return CacheKeyInfo{
				TTL:         TTLMedium,
				Invalidates: []string{},
			}
		}
		
	case "topology":
		return CacheKeyInfo{
			TTL:         TTLShort, // Topology view should be relatively fresh
			Invalidates: []string{},
		}
		
	default:
		return CacheKeyInfo{
			TTL:         TTLShort,
			Invalidates: []string{},
		}
	}
	
	// Default case
	return CacheKeyInfo{
		TTL:         TTLShort,
		Invalidates: []string{},
	}
}

// Batch invalidation helpers

// InvalidateSwitch invalidates all cache entries related to a switch
func InvalidateSwitch(cache Cache, switchUUID string) error {
	patterns := []string{
		SwitchKey(switchUUID),
		SwitchPattern(),
		PortsByParentPattern(switchUUID),
		TopologyPattern(),
	}
	
	ctx := context.Background()
	for _, pattern := range patterns {
		if err := cache.Clear(ctx, pattern); err != nil {
			return err
		}
	}
	
	return nil
}

// InvalidateRouter invalidates all cache entries related to a router
func InvalidateRouter(cache Cache, routerUUID string) error {
	patterns := []string{
		RouterKey(routerUUID),
		RouterPattern(),
		PortsByParentPattern(routerUUID),
		NATListKey(routerUUID),
		TopologyPattern(),
	}
	
	ctx := context.Background()
	for _, pattern := range patterns {
		if err := cache.Clear(ctx, pattern); err != nil {
			return err
		}
	}
	
	return nil
}

// InvalidateTopology invalidates all topology-related cache entries
func InvalidateTopology(cache Cache) error {
	return cache.Clear(context.Background(), TopologyPattern())
}