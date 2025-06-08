package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SessionStore manages WebSocket session affinity
type SessionStore struct {
	redis    *redis.Client
	nodeID   string
	logger   *zap.Logger
	mu       sync.RWMutex
	local    map[string]*SessionInfo
	ttl      time.Duration
}

// SessionInfo contains information about a WebSocket session
type SessionInfo struct {
	SessionID    string    `json:"session_id"`
	UserID       string    `json:"user_id"`
	NodeID       string    `json:"node_id"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastActivity time.Time `json:"last_activity"`
	Metadata     map[string]string `json:"metadata"`
}

// SessionStoreConfig holds configuration for session store
type SessionStoreConfig struct {
	NodeID         string
	SessionTTL     time.Duration
	CleanupInterval time.Duration
	RedisKeyPrefix  string
}

// DefaultSessionStoreConfig returns default configuration
func DefaultSessionStoreConfig() *SessionStoreConfig {
	return &SessionStoreConfig{
		SessionTTL:      30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
		RedisKeyPrefix:  "ovncp:session:",
	}
}

// NewSessionStore creates a new session store
func NewSessionStore(cfg *SessionStoreConfig, redis *redis.Client, logger *zap.Logger) *SessionStore {
	store := &SessionStore{
		redis:  redis,
		nodeID: cfg.NodeID,
		logger: logger,
		local:  make(map[string]*SessionInfo),
		ttl:    cfg.SessionTTL,
	}

	// Start cleanup goroutine
	go store.cleanupLoop(cfg.CleanupInterval)

	return store
}

// Register registers a new WebSocket session
func (s *SessionStore) Register(ctx context.Context, sessionID, userID string) error {
	info := &SessionInfo{
		SessionID:    sessionID,
		UserID:       userID,
		NodeID:       s.nodeID,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		Metadata:     make(map[string]string),
	}

	// Store locally
	s.mu.Lock()
	s.local[sessionID] = info
	s.mu.Unlock()

	// Store in Redis for cluster visibility
	key := s.sessionKey(sessionID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal session info: %w", err)
	}

	if err := s.redis.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session in Redis: %w", err)
	}

	// Also store user->sessions mapping
	userKey := s.userSessionKey(userID)
	if err := s.redis.SAdd(ctx, userKey, sessionID).Err(); err != nil {
		s.logger.Warn("Failed to add session to user set", zap.Error(err))
	}
	s.redis.Expire(ctx, userKey, s.ttl)

	s.logger.Debug("Session registered",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("node_id", s.nodeID))

	return nil
}

// Unregister removes a WebSocket session
func (s *SessionStore) Unregister(ctx context.Context, sessionID string) error {
	// Get session info for cleanup
	info, err := s.GetSession(ctx, sessionID)
	if err == nil && info != nil {
		// Remove from user->sessions mapping
		userKey := s.userSessionKey(info.UserID)
		s.redis.SRem(ctx, userKey, sessionID)
	}

	// Remove from local store
	s.mu.Lock()
	delete(s.local, sessionID)
	s.mu.Unlock()

	// Remove from Redis
	key := s.sessionKey(sessionID)
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove session from Redis: %w", err)
	}

	s.logger.Debug("Session unregistered", zap.String("session_id", sessionID))
	return nil
}

// UpdateActivity updates the last activity time for a session
func (s *SessionStore) UpdateActivity(ctx context.Context, sessionID string) error {
	// Update local store
	s.mu.Lock()
	if info, exists := s.local[sessionID]; exists {
		info.LastActivity = time.Now()
	}
	s.mu.Unlock()

	// Update in Redis
	info, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	info.LastActivity = time.Now()
	
	key := s.sessionKey(sessionID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal session info: %w", err)
	}

	if err := s.redis.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to update session in Redis: %w", err)
	}

	return nil
}

// GetSession retrieves session information
func (s *SessionStore) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	// Check local store first
	s.mu.RLock()
	if info, exists := s.local[sessionID]; exists {
		s.mu.RUnlock()
		return info, nil
	}
	s.mu.RUnlock()

	// Check Redis
	key := s.sessionKey(sessionID)
	data, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	var info SessionInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session info: %w", err)
	}

	return &info, nil
}

// GetNodeForSession returns the node ID where a session is connected
func (s *SessionStore) GetNodeForSession(ctx context.Context, sessionID string) (string, error) {
	info, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return info.NodeID, nil
}

// GetSessionsForUser returns all sessions for a user
func (s *SessionStore) GetSessionsForUser(ctx context.Context, userID string) ([]*SessionInfo, error) {
	userKey := s.userSessionKey(userID)
	
	sessionIDs, err := s.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	sessions := make([]*SessionInfo, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		info, err := s.GetSession(ctx, sessionID)
		if err != nil {
			// Session might have expired
			continue
		}
		sessions = append(sessions, info)
	}

	return sessions, nil
}

// GetLocalSessions returns all sessions connected to this node
func (s *SessionStore) GetLocalSessions() []*SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*SessionInfo, 0, len(s.local))
	for _, info := range s.local {
		sessions = append(sessions, info)
	}
	return sessions
}

// IsLocal checks if a session is connected to this node
func (s *SessionStore) IsLocal(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.local[sessionID]
	return exists
}

// SetMetadata sets metadata for a session
func (s *SessionStore) SetMetadata(ctx context.Context, sessionID string, key, value string) error {
	info, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if info.Metadata == nil {
		info.Metadata = make(map[string]string)
	}
	info.Metadata[key] = value

	// Update local store if applicable
	s.mu.Lock()
	if localInfo, exists := s.local[sessionID]; exists {
		localInfo.Metadata[key] = value
	}
	s.mu.Unlock()

	// Update in Redis
	redisKey := s.sessionKey(sessionID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal session info: %w", err)
	}

	if err := s.redis.Set(ctx, redisKey, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to update session metadata: %w", err)
	}

	return nil
}

// cleanupLoop periodically cleans up expired sessions
func (s *SessionStore) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes expired sessions
func (s *SessionStore) cleanup() {
	_ = context.Background()
	
	// Clean up local sessions
	s.mu.Lock()
	for sessionID, info := range s.local {
		if time.Since(info.LastActivity) > s.ttl {
			delete(s.local, sessionID)
			s.logger.Debug("Cleaned up inactive local session", zap.String("session_id", sessionID))
		}
	}
	s.mu.Unlock()

	// Note: Redis sessions are automatically cleaned up by TTL
}

// sessionKey returns the Redis key for a session
func (s *SessionStore) sessionKey(sessionID string) string {
	return fmt.Sprintf("ovncp:session:%s", sessionID)
}

// userSessionKey returns the Redis key for user's sessions
func (s *SessionStore) userSessionKey(userID string) string {
	return fmt.Sprintf("ovncp:user_sessions:%s", userID)
}

// MigrateSession migrates a session to another node
func (s *SessionStore) MigrateSession(ctx context.Context, sessionID string, targetNodeID string) error {
	info, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Update node assignment
	info.NodeID = targetNodeID

	// Remove from local store if present
	s.mu.Lock()
	delete(s.local, sessionID)
	s.mu.Unlock()

	// Update in Redis
	key := s.sessionKey(sessionID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal session info: %w", err)
	}

	if err := s.redis.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to migrate session: %w", err)
	}

	s.logger.Info("Session migrated",
		zap.String("session_id", sessionID),
		zap.String("from_node", s.nodeID),
		zap.String("to_node", targetNodeID))

	return nil
}