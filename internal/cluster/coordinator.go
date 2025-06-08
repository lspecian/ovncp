package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NodeInfo represents information about a cluster node
type NodeInfo struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	IP            string            `json:"ip"`
	Port          int               `json:"port"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	Status        string            `json:"status"`
	Metadata      map[string]string `json:"metadata"`
}

// Coordinator manages cluster coordination
type Coordinator struct {
	nodeID       string
	nodeInfo     *NodeInfo
	redis        *redis.Client
	logger       *zap.Logger
	mu           sync.RWMutex
	nodes        map[string]*NodeInfo
	eventHandlers map[string][]EventHandler
	stopCh       chan struct{}
}

// EventType represents cluster event types
type EventType string

const (
	EventNodeJoin    EventType = "node_join"
	EventNodeLeave   EventType = "node_leave"
	EventNodeUpdate  EventType = "node_update"
	EventCacheInvalidate EventType = "cache_invalidate"
	EventConfigUpdate    EventType = "config_update"
)

// Event represents a cluster event
type Event struct {
	Type      EventType              `json:"type"`
	NodeID    string                 `json:"node_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// EventHandler handles cluster events
type EventHandler func(event *Event)

// CoordinatorConfig holds configuration for cluster coordinator
type CoordinatorConfig struct {
	NodeID          string
	Hostname        string
	IP              string
	Port            int
	HeartbeatInterval time.Duration
	NodeTimeout       time.Duration
	RedisKeyPrefix    string
}

// DefaultCoordinatorConfig returns default configuration
func DefaultCoordinatorConfig() *CoordinatorConfig {
	return &CoordinatorConfig{
		NodeID:            uuid.New().String(),
		HeartbeatInterval: 5 * time.Second,
		NodeTimeout:       30 * time.Second,
		RedisKeyPrefix:    "ovncp:cluster:",
	}
}

// NewCoordinator creates a new cluster coordinator
func NewCoordinator(cfg *CoordinatorConfig, redis *redis.Client, logger *zap.Logger) *Coordinator {
	nodeInfo := &NodeInfo{
		ID:       cfg.NodeID,
		Hostname: cfg.Hostname,
		IP:       cfg.IP,
		Port:     cfg.Port,
		Status:   "active",
		Metadata: make(map[string]string),
	}

	return &Coordinator{
		nodeID:        cfg.NodeID,
		nodeInfo:      nodeInfo,
		redis:         redis,
		logger:        logger,
		nodes:         make(map[string]*NodeInfo),
		eventHandlers: make(map[string][]EventHandler),
		stopCh:        make(chan struct{}),
	}
}

// Start starts the cluster coordinator
func (c *Coordinator) Start(ctx context.Context) error {
	c.logger.Info("Starting cluster coordinator", zap.String("node_id", c.nodeID))

	// Register this node
	if err := c.registerNode(ctx); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	// Start heartbeat
	go c.heartbeatLoop(ctx)

	// Start node discovery
	go c.discoveryLoop(ctx)

	// Start event listener
	go c.eventLoop(ctx)

	// Emit node join event
	c.publishEvent(&Event{
		Type:      EventNodeJoin,
		NodeID:    c.nodeID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"node_info": c.nodeInfo,
		},
	})

	return nil
}

// Stop stops the cluster coordinator
func (c *Coordinator) Stop() {
	c.logger.Info("Stopping cluster coordinator")
	
	// Emit node leave event
	c.publishEvent(&Event{
		Type:      EventNodeLeave,
		NodeID:    c.nodeID,
		Timestamp: time.Now(),
	})

	// Unregister node
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.unregisterNode(ctx)

	close(c.stopCh)
}

// GetNodeID returns the current node ID
func (c *Coordinator) GetNodeID() string {
	return c.nodeID
}

// GetNodes returns all known cluster nodes
func (c *Coordinator) GetNodes() []*NodeInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetActiveNodes returns only active cluster nodes
func (c *Coordinator) GetActiveNodes() []*NodeInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]*NodeInfo, 0)
	for _, node := range c.nodes {
		if node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// RegisterEventHandler registers a handler for cluster events
func (c *Coordinator) RegisterEventHandler(eventType EventType, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := string(eventType)
	c.eventHandlers[key] = append(c.eventHandlers[key], handler)
}

// PublishCacheInvalidation publishes a cache invalidation event
func (c *Coordinator) PublishCacheInvalidation(patterns []string) {
	c.publishEvent(&Event{
		Type:      EventCacheInvalidate,
		NodeID:    c.nodeID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"patterns": patterns,
		},
	})
}

// registerNode registers this node in the cluster
func (c *Coordinator) registerNode(ctx context.Context) error {
	c.nodeInfo.LastHeartbeat = time.Now()
	
	data, err := json.Marshal(c.nodeInfo)
	if err != nil {
		return err
	}

	key := c.nodeKey(c.nodeID)
	ttl := 30 * time.Second

	return c.redis.Set(ctx, key, data, ttl).Err()
}

// unregisterNode removes this node from the cluster
func (c *Coordinator) unregisterNode(ctx context.Context) error {
	key := c.nodeKey(c.nodeID)
	return c.redis.Del(ctx, key).Err()
}

// heartbeatLoop sends periodic heartbeats
func (c *Coordinator) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.registerNode(ctx); err != nil {
				c.logger.Error("Failed to send heartbeat", zap.Error(err))
			}
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// discoveryLoop discovers other nodes in the cluster
func (c *Coordinator) discoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Initial discovery
	c.discoverNodes(ctx)

	for {
		select {
		case <-ticker.C:
			c.discoverNodes(ctx)
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// discoverNodes discovers all nodes in the cluster
func (c *Coordinator) discoverNodes(ctx context.Context) {
	pattern := "ovncp:cluster:node:*"
	
	var cursor uint64
	discovered := make(map[string]bool)

	for {
		keys, nextCursor, err := c.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.logger.Error("Failed to scan for nodes", zap.Error(err))
			return
		}

		for _, key := range keys {
			nodeID := c.nodeIDFromKey(key)
			if nodeID == c.nodeID {
				continue // Skip self
			}

			discovered[nodeID] = true

			// Get node info
			data, err := c.redis.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var nodeInfo NodeInfo
			if err := json.Unmarshal([]byte(data), &nodeInfo); err != nil {
				continue
			}

			c.updateNode(&nodeInfo)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	// Remove nodes that are no longer discovered
	c.mu.Lock()
	for nodeID := range c.nodes {
		if !discovered[nodeID] {
			delete(c.nodes, nodeID)
			c.logger.Info("Node removed from cluster", zap.String("node_id", nodeID))
		}
	}
	c.mu.Unlock()
}

// updateNode updates node information
func (c *Coordinator) updateNode(nodeInfo *NodeInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existing, exists := c.nodes[nodeInfo.ID]
	if !exists {
		c.logger.Info("New node joined cluster", zap.String("node_id", nodeInfo.ID))
	}

	// Check if node is still active
	if time.Since(nodeInfo.LastHeartbeat) > 30*time.Second {
		nodeInfo.Status = "inactive"
	}

	c.nodes[nodeInfo.ID] = nodeInfo

	// Emit event if status changed
	if exists && existing.Status != nodeInfo.Status {
		c.publishEvent(&Event{
			Type:      EventNodeUpdate,
			NodeID:    nodeInfo.ID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"old_status": existing.Status,
				"new_status": nodeInfo.Status,
			},
		})
	}
}

// eventLoop listens for cluster events
func (c *Coordinator) eventLoop(ctx context.Context) {
	pubsub := c.redis.Subscribe(ctx, "ovncp:cluster:events")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				c.logger.Error("Failed to unmarshal event", zap.Error(err))
				continue
			}

			// Skip events from self
			if event.NodeID == c.nodeID {
				continue
			}

			c.handleEvent(&event)
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// handleEvent processes a cluster event
func (c *Coordinator) handleEvent(event *Event) {
	c.mu.RLock()
	handlers := c.eventHandlers[string(event.Type)]
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// publishEvent publishes an event to the cluster
func (c *Coordinator) publishEvent(event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		c.logger.Error("Failed to marshal event", zap.Error(err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.redis.Publish(ctx, "ovncp:cluster:events", data).Err(); err != nil {
		c.logger.Error("Failed to publish event", zap.Error(err))
	}
}

// nodeKey returns the Redis key for a node
func (c *Coordinator) nodeKey(nodeID string) string {
	return fmt.Sprintf("ovncp:cluster:node:%s", nodeID)
}

// nodeIDFromKey extracts node ID from Redis key
func (c *Coordinator) nodeIDFromKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

// IsLeader checks if this node is the cluster leader
func (c *Coordinator) IsLeader() bool {
	nodes := c.GetActiveNodes()
	if len(nodes) == 0 {
		return true // No other nodes, we're the leader
	}

	// Simple leader election: node with lowest ID is leader
	lowestID := c.nodeID
	for _, node := range nodes {
		if node.ID < lowestID {
			lowestID = node.ID
		}
	}

	return lowestID == c.nodeID
}

// SelectNode selects a node for handling a request (for load balancing)
func (c *Coordinator) SelectNode(key string) *NodeInfo {
	nodes := c.GetActiveNodes()
	if len(nodes) == 0 {
		return c.nodeInfo // Return self if no other nodes
	}

	// Simple consistent hashing
	hash := 0
	for _, b := range []byte(key) {
		hash = hash*31 + int(b)
	}

	index := hash % len(nodes)
	return nodes[index]
}