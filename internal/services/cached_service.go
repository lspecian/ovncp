package services

import (
	"context"
	"fmt"

	"github.com/lspecian/ovncp/internal/cache"
	"github.com/lspecian/ovncp/internal/models"
	"go.uber.org/zap"
)

// CachedOVNService wraps OVNService with caching
type CachedOVNService struct {
	service OVNServiceInterface
	cache   cache.Cache
	logger  *zap.Logger
}

// NewCachedOVNService creates a new cached OVN service
func NewCachedOVNService(service OVNServiceInterface, cache cache.Cache, logger *zap.Logger) *CachedOVNService {
	return &CachedOVNService{
		service: service,
		cache:   cache,
		logger:  logger,
	}
}

// Logical Switch operations with caching

func (s *CachedOVNService) ListLogicalSwitches(ctx context.Context) ([]*models.LogicalSwitch, error) {
	// Generate cache key
	cacheKey := cache.SwitchListKey(0, 0, nil)
	
	// Try to get from cache
	var switches []*models.LogicalSwitch
	err := s.cache.Get(ctx, cacheKey, &switches)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return switches, nil
	}
	
	// Cache miss, get from service
	switches, err = s.service.ListLogicalSwitches(ctx)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("switch", "list")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, switches, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache switches", zap.Error(err))
		}
	}
	
	return switches, nil
}

func (s *CachedOVNService) GetLogicalSwitch(ctx context.Context, id string) (*models.LogicalSwitch, error) {
	// Generate cache key
	cacheKey := cache.SwitchKey(id)
	
	// Try to get from cache
	var sw models.LogicalSwitch
	err := s.cache.Get(ctx, cacheKey, &sw)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return &sw, nil
	}
	
	// Cache miss, get from service
	switchPtr, err := s.service.GetLogicalSwitch(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("switch", "get")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, switchPtr, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache switch", zap.Error(err))
		}
	}
	
	return switchPtr, nil
}

func (s *CachedOVNService) CreateLogicalSwitch(ctx context.Context, sw *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	// Create the switch
	createdSwitch, err := s.service.CreateLogicalSwitch(ctx, sw)
	if err != nil {
		return nil, err
	}
	
	// Invalidate related caches
	keyInfo := cache.GetCacheKeyInfo("switch", "create")
	for _, pattern := range keyInfo.Invalidates {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			s.logger.Warn("Failed to invalidate cache", zap.String("pattern", pattern), zap.Error(err))
		}
	}
	
	return createdSwitch, nil
}

func (s *CachedOVNService) UpdateLogicalSwitch(ctx context.Context, id string, sw *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	// Update the switch
	updatedSwitch, err := s.service.UpdateLogicalSwitch(ctx, id, sw)
	if err != nil {
		return nil, err
	}
	
	// Invalidate specific switch cache and related patterns
	if err := cache.InvalidateSwitch(s.cache, id); err != nil {
		s.logger.Warn("Failed to invalidate switch cache", zap.Error(err))
	}
	
	return updatedSwitch, nil
}

func (s *CachedOVNService) DeleteLogicalSwitch(ctx context.Context, id string) error {
	// Delete the switch
	err := s.service.DeleteLogicalSwitch(ctx, id)
	if err != nil {
		return err
	}
	
	// Invalidate related caches
	if err := cache.InvalidateSwitch(s.cache, id); err != nil {
		s.logger.Warn("Failed to invalidate switch cache", zap.Error(err))
	}
	
	return nil
}

// Logical Router operations with caching

func (s *CachedOVNService) ListLogicalRouters(ctx context.Context) ([]*models.LogicalRouter, error) {
	// Generate cache key
	cacheKey := cache.RouterListKey(0, 0, nil)
	
	// Try to get from cache
	var routers []*models.LogicalRouter
	err := s.cache.Get(ctx, cacheKey, &routers)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return routers, nil
	}
	
	// Cache miss, get from service
	routers, err = s.service.ListLogicalRouters(ctx)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("router", "list")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, routers, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache routers", zap.Error(err))
		}
	}
	
	return routers, nil
}

func (s *CachedOVNService) GetLogicalRouter(ctx context.Context, id string) (*models.LogicalRouter, error) {
	// Generate cache key
	cacheKey := cache.RouterKey(id)
	
	// Try to get from cache
	var router models.LogicalRouter
	err := s.cache.Get(ctx, cacheKey, &router)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return &router, nil
	}
	
	// Cache miss, get from service
	routerPtr, err := s.service.GetLogicalRouter(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("router", "get")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, routerPtr, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache router", zap.Error(err))
		}
	}
	
	return routerPtr, nil
}

func (s *CachedOVNService) CreateLogicalRouter(ctx context.Context, router *models.LogicalRouter) (*models.LogicalRouter, error) {
	// Create the router
	createdRouter, err := s.service.CreateLogicalRouter(ctx, router)
	if err != nil {
		return nil, err
	}
	
	// Invalidate related caches
	keyInfo := cache.GetCacheKeyInfo("router", "create")
	for _, pattern := range keyInfo.Invalidates {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			s.logger.Warn("Failed to invalidate cache", zap.String("pattern", pattern), zap.Error(err))
		}
	}
	
	return createdRouter, nil
}

func (s *CachedOVNService) UpdateLogicalRouter(ctx context.Context, id string, router *models.LogicalRouter) (*models.LogicalRouter, error) {
	// Update the router
	updatedRouter, err := s.service.UpdateLogicalRouter(ctx, id, router)
	if err != nil {
		return nil, err
	}
	
	// Invalidate specific router cache and related patterns
	if err := cache.InvalidateRouter(s.cache, id); err != nil {
		s.logger.Warn("Failed to invalidate router cache", zap.Error(err))
	}
	
	return updatedRouter, nil
}

func (s *CachedOVNService) DeleteLogicalRouter(ctx context.Context, id string) error {
	// Delete the router
	err := s.service.DeleteLogicalRouter(ctx, id)
	if err != nil {
		return err
	}
	
	// Invalidate related caches
	if err := cache.InvalidateRouter(s.cache, id); err != nil {
		s.logger.Warn("Failed to invalidate router cache", zap.Error(err))
	}
	
	return nil
}

// Port operations with caching

func (s *CachedOVNService) ListPorts(ctx context.Context, switchID string) ([]*models.LogicalSwitchPort, error) {
	// Generate cache key
	cacheKey := cache.PortListKey(switchID, "switch")
	
	// Try to get from cache
	var ports []*models.LogicalSwitchPort
	err := s.cache.Get(ctx, cacheKey, &ports)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return ports, nil
	}
	
	// Cache miss, get from service
	ports, err = s.service.ListPorts(ctx, switchID)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("port", "list")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, ports, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache ports", zap.Error(err))
		}
	}
	
	return ports, nil
}

func (s *CachedOVNService) GetPort(ctx context.Context, id string) (*models.LogicalSwitchPort, error) {
	// Generate cache key
	cacheKey := cache.PortKey(id)
	
	// Try to get from cache
	var port models.LogicalSwitchPort
	err := s.cache.Get(ctx, cacheKey, &port)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return &port, nil
	}
	
	// Cache miss, get from service
	portPtr, err := s.service.GetPort(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("port", "get")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, portPtr, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache port", zap.Error(err))
		}
	}
	
	return portPtr, nil
}

func (s *CachedOVNService) CreatePort(ctx context.Context, switchID string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	// Create the port
	createdPort, err := s.service.CreatePort(ctx, switchID, port)
	if err != nil {
		return nil, err
	}
	
	// Invalidate related caches
	keyInfo := cache.GetCacheKeyInfo("port", "create")
	for _, pattern := range keyInfo.Invalidates {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			s.logger.Warn("Failed to invalidate cache", zap.String("pattern", pattern), zap.Error(err))
		}
	}
	
	// Also invalidate parent's port list
	portListKey := cache.PortListKey(switchID, "switch")
	if err := s.cache.Delete(ctx, portListKey); err != nil {
		s.logger.Warn("Failed to invalidate port list cache", zap.Error(err))
	}
	
	return createdPort, nil
}

func (s *CachedOVNService) UpdatePort(ctx context.Context, id string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	// Update the port
	updatedPort, err := s.service.UpdatePort(ctx, id, port)
	if err != nil {
		return nil, err
	}
	
	// Invalidate specific port cache
	if err := s.cache.Delete(ctx, cache.PortKey(id)); err != nil {
		s.logger.Warn("Failed to invalidate port cache", zap.Error(err))
	}
	
	// Invalidate parent's port list
	if updatedPort.ParentUUID != "" {
		portListKey := cache.PortListKey(updatedPort.ParentUUID, updatedPort.ParentType)
		if err := s.cache.Delete(ctx, portListKey); err != nil {
			s.logger.Warn("Failed to invalidate port list cache", zap.Error(err))
		}
	}
	
	// Invalidate topology
	if err := cache.InvalidateTopology(s.cache); err != nil {
		s.logger.Warn("Failed to invalidate topology cache", zap.Error(err))
	}
	
	return updatedPort, nil
}

func (s *CachedOVNService) DeletePort(ctx context.Context, id string) error {
	// Get port info before deletion for cache invalidation
	port, _ := s.GetPort(ctx, id)
	
	// Delete the port
	err := s.service.DeletePort(ctx, id)
	if err != nil {
		return err
	}
	
	// Invalidate port cache
	if err := s.cache.Delete(ctx, cache.PortKey(id)); err != nil {
		s.logger.Warn("Failed to invalidate port cache", zap.Error(err))
	}
	
	// Invalidate parent's port list if we have the info
	if port != nil && port.ParentUUID != "" {
		portListKey := cache.PortListKey(port.ParentUUID, port.ParentType)
		if err := s.cache.Delete(ctx, portListKey); err != nil {
			s.logger.Warn("Failed to invalidate port list cache", zap.Error(err))
		}
	}
	
	// Invalidate topology
	if err := cache.InvalidateTopology(s.cache); err != nil {
		s.logger.Warn("Failed to invalidate topology cache", zap.Error(err))
	}
	
	return nil
}

// ACL operations with caching

func (s *CachedOVNService) ListACLs(ctx context.Context, switchID string) ([]*models.ACL, error) {
	// Generate cache key
	cacheKey := cache.ACLListKey(map[string]string{"switch": switchID})
	
	// Try to get from cache
	var acls []*models.ACL
	err := s.cache.Get(ctx, cacheKey, &acls)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return acls, nil
	}
	
	// Cache miss, get from service
	acls, err = s.service.ListACLs(ctx, switchID)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("acl", "list")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, acls, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache ACLs", zap.Error(err))
		}
	}
	
	return acls, nil
}

func (s *CachedOVNService) GetACL(ctx context.Context, id string) (*models.ACL, error) {
	// Generate cache key
	cacheKey := cache.ACLKey(id)
	
	// Try to get from cache
	var acl models.ACL
	err := s.cache.Get(ctx, cacheKey, &acl)
	if err == nil {
		s.logger.Debug("Cache hit", zap.String("key", cacheKey))
		return &acl, nil
	}
	
	// Cache miss, get from service
	aclPtr, err := s.service.GetACL(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("acl", "get")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, aclPtr, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache ACL", zap.Error(err))
		}
	}
	
	return aclPtr, nil
}

func (s *CachedOVNService) CreateACL(ctx context.Context, switchID string, acl *models.ACL) (*models.ACL, error) {
	// Create the ACL
	createdACL, err := s.service.CreateACL(ctx, switchID, acl)
	if err != nil {
		return nil, err
	}
	
	// Invalidate related caches
	keyInfo := cache.GetCacheKeyInfo("acl", "create")
	for _, pattern := range keyInfo.Invalidates {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			s.logger.Warn("Failed to invalidate cache", zap.String("pattern", pattern), zap.Error(err))
		}
	}
	
	return createdACL, nil
}

func (s *CachedOVNService) UpdateACL(ctx context.Context, id string, acl *models.ACL) (*models.ACL, error) {
	// Update the ACL
	updatedACL, err := s.service.UpdateACL(ctx, id, acl)
	if err != nil {
		return nil, err
	}
	
	// Invalidate specific ACL cache and lists
	if err := s.cache.Delete(ctx, cache.ACLKey(id)); err != nil {
		s.logger.Warn("Failed to invalidate ACL cache", zap.Error(err))
	}
	
	// Clear all ACL lists as they might be filtered
	if err := s.cache.Clear(ctx, cache.ACLPattern()); err != nil {
		s.logger.Warn("Failed to invalidate ACL pattern", zap.Error(err))
	}
	
	return updatedACL, nil
}

func (s *CachedOVNService) DeleteACL(ctx context.Context, id string) error {
	// Delete the ACL
	err := s.service.DeleteACL(ctx, id)
	if err != nil {
		return err
	}
	
	// Invalidate ACL caches
	if err := s.cache.Delete(ctx, cache.ACLKey(id)); err != nil {
		s.logger.Warn("Failed to invalidate ACL cache", zap.Error(err))
	}
	
	// Clear all ACL lists
	if err := s.cache.Clear(ctx, cache.ACLPattern()); err != nil {
		s.logger.Warn("Failed to invalidate ACL pattern", zap.Error(err))
	}
	
	return nil
}

// Topology operation with caching

func (s *CachedOVNService) GetTopology(ctx context.Context) (*Topology, error) {
	// Generate cache key
	cacheKey := cache.TopologyKey()
	
	// Try to get from cache
	var topology Topology
	err := s.cache.Get(ctx, cacheKey, &topology)
	if err == nil {
		s.logger.Debug("Cache hit for topology")
		return &topology, nil
	}
	
	// Cache miss, get from service
	topologyPtr, err := s.service.GetTopology(ctx)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	keyInfo := cache.GetCacheKeyInfo("topology", "get")
	if keyInfo.TTL > 0 {
		if err := s.cache.Set(ctx, cacheKey, topologyPtr, keyInfo.TTL); err != nil {
			s.logger.Warn("Failed to cache topology", zap.Error(err))
		}
	}
	
	return topologyPtr, nil
}

// Transaction executes multiple operations atomically (no caching for transactions)
func (s *CachedOVNService) ExecuteTransaction(ctx context.Context, ops []TransactionOp) error {
	// Execute transaction
	err := s.service.ExecuteTransaction(ctx, ops)
	if err != nil {
		return err
	}
	
	// Invalidate all caches affected by the transaction
	// This is a conservative approach - we clear more than necessary
	patterns := make(map[string]bool)
	
	for _, op := range ops {
		switch op.ResourceType {
		case "logical_switch":
			patterns[cache.SwitchPattern()] = true
			patterns[cache.TopologyPattern()] = true
		case "logical_router":
			patterns[cache.RouterPattern()] = true
			patterns[cache.TopologyPattern()] = true
		case "logical_port":
			patterns[cache.PortPattern()] = true
			patterns[cache.TopologyPattern()] = true
		case "acl":
			patterns[cache.ACLPattern()] = true
		}
	}
	
	// Clear all affected patterns
	for pattern := range patterns {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			s.logger.Warn("Failed to invalidate cache after transaction",
				zap.String("pattern", pattern),
				zap.Error(err))
		}
	}
	
	return nil
}

// Cache management methods

// WarmCache pre-populates cache with frequently accessed data
func (s *CachedOVNService) WarmCache(ctx context.Context) error {
	s.logger.Info("Warming cache")
	
	// Warm up topology cache
	if _, err := s.GetTopology(ctx); err != nil {
		s.logger.Warn("Failed to warm topology cache", zap.Error(err))
	}
	
	// Warm up switch list cache
	if _, err := s.ListLogicalSwitches(ctx); err != nil {
		s.logger.Warn("Failed to warm switch cache", zap.Error(err))
	}
	
	// Warm up router list cache
	if _, err := s.ListLogicalRouters(ctx); err != nil {
		s.logger.Warn("Failed to warm router cache", zap.Error(err))
	}
	
	s.logger.Info("Cache warming completed")
	return nil
}

// ClearCache clears all cached data
func (s *CachedOVNService) ClearCache(ctx context.Context) error {
	patterns := []string{
		cache.SwitchPattern(),
		cache.RouterPattern(),
		cache.PortPattern(),
		cache.ACLPattern(),
		cache.TopologyPattern(),
		cache.LoadBalancerKey("*"),
		cache.NATKey("*"),
	}
	
	for _, pattern := range patterns {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			return fmt.Errorf("failed to clear cache pattern %s: %w", pattern, err)
		}
	}
	
	s.logger.Info("Cache cleared")
	return nil
}

// GetCacheStats returns cache statistics
func (s *CachedOVNService) GetCacheStats() cache.CacheStats {
	if rc, ok := s.cache.(*cache.RedisCache); ok {
		return rc.Stats()
	}
	if mc, ok := s.cache.(*cache.MemoryCache); ok {
		return mc.Stats()
	}
	return cache.CacheStats{}
}