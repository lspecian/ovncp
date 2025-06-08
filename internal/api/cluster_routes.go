package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/cluster"
)

// ClusterHandler handles cluster-related endpoints
type ClusterHandler struct {
	coordinator  *cluster.Coordinator
	sessionStore *cluster.SessionStore
	lockManager  *cluster.LockManager
}

// NewClusterHandler creates a new cluster handler
func NewClusterHandler(coordinator *cluster.Coordinator, sessionStore *cluster.SessionStore, lockManager *cluster.LockManager) *ClusterHandler {
	return &ClusterHandler{
		coordinator:  coordinator,
		sessionStore: sessionStore,
		lockManager:  lockManager,
	}
}

// RegisterClusterRoutes registers cluster management routes
func (h *ClusterHandler) RegisterClusterRoutes(router *gin.RouterGroup) {
	cluster := router.Group("/cluster")
	{
		cluster.GET("/nodes", h.getNodes)
		cluster.GET("/nodes/:id", h.getNode)
		cluster.GET("/leader", h.getLeader)
		cluster.GET("/sessions", h.getSessions)
		cluster.GET("/sessions/:id", h.getSession)
		cluster.GET("/sessions/user/:userID", h.getUserSessions)
		cluster.POST("/sessions/:id/migrate", h.migrateSession)
		cluster.GET("/locks", h.getLocks)
		cluster.DELETE("/locks/:key", h.releaseLock)
	}
}

// getNodes returns all cluster nodes
func (h *ClusterHandler) getNodes(c *gin.Context) {
	activeOnly := c.Query("active") == "true"
	
	var nodes []*cluster.NodeInfo
	if activeOnly {
		nodes = h.coordinator.GetActiveNodes()
	} else {
		nodes = h.coordinator.GetNodes()
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"count": len(nodes),
		"self":  h.coordinator.GetNodeID(),
	})
}

// getNode returns information about a specific node
func (h *ClusterHandler) getNode(c *gin.Context) {
	nodeID := c.Param("id")
	
	nodes := h.coordinator.GetNodes()
	for _, node := range nodes {
		if node.ID == nodeID {
			c.JSON(http.StatusOK, node)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "node not found",
	})
}

// getLeader returns the current cluster leader
func (h *ClusterHandler) getLeader(c *gin.Context) {
	isLeader := h.coordinator.IsLeader()
	
	response := gin.H{
		"node_id":   h.coordinator.GetNodeID(),
		"is_leader": isLeader,
	}

	// Find the actual leader if we're not it
	if !isLeader {
		nodes := h.coordinator.GetActiveNodes()
		var leaderID string
		for _, node := range nodes {
			if leaderID == "" || node.ID < leaderID {
				leaderID = node.ID
			}
		}
		response["leader_id"] = leaderID
	}

	c.JSON(http.StatusOK, response)
}

// getSessions returns all WebSocket sessions
func (h *ClusterHandler) getSessions(c *gin.Context) {
	localOnly := c.Query("local") == "true"
	
	if localOnly {
		sessions := h.sessionStore.GetLocalSessions()
		c.JSON(http.StatusOK, gin.H{
			"sessions": sessions,
			"count":    len(sessions),
			"node_id":  h.coordinator.GetNodeID(),
		})
		return
	}

	// For all sessions, we'd need to aggregate from all nodes
	// For now, just return local sessions
	sessions := h.sessionStore.GetLocalSessions()
	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
		"node_id":  h.coordinator.GetNodeID(),
		"note":     "showing local sessions only",
	})
}

// getSession returns information about a specific session
func (h *ClusterHandler) getSession(c *gin.Context) {
	sessionID := c.Param("id")
	
	session, err := h.sessionStore.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "session not found",
		})
		return
	}

	c.JSON(http.StatusOK, session)
}

// getUserSessions returns all sessions for a user
func (h *ClusterHandler) getUserSessions(c *gin.Context) {
	userID := c.Param("userID")
	
	sessions, err := h.sessionStore.GetSessionsForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// migrateSession migrates a session to another node
func (h *ClusterHandler) migrateSession(c *gin.Context) {
	sessionID := c.Param("id")
	
	var req struct {
		TargetNodeID string `json:"target_node_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Verify target node exists and is active
	found := false
	for _, node := range h.coordinator.GetActiveNodes() {
		if node.ID == req.TargetNodeID {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "target node not found or not active",
		})
		return
	}

	err := h.sessionStore.MigrateSession(c.Request.Context(), sessionID, req.TargetNodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "session migrated successfully",
		"session_id": sessionID,
		"target_node_id": req.TargetNodeID,
	})
}

// getLocks returns information about distributed locks
func (h *ClusterHandler) getLocks(c *gin.Context) {
	// This would require tracking locks in Redis
	// For now, return a placeholder
	c.JSON(http.StatusOK, gin.H{
		"message": "lock information not implemented",
		"node_id": h.coordinator.GetNodeID(),
	})
}

// releaseLock releases a distributed lock
func (h *ClusterHandler) releaseLock(c *gin.Context) {
	key := c.Param("key")
	
	err := h.lockManager.ReleaseLock(c.Request.Context(), key)
	if err != nil {
		if err == cluster.ErrLockNotHeld {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "lock not held by this node",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "lock released successfully",
		"key": key,
	})
}