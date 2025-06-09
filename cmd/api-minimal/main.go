package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS middleware for frontend
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "ovncp-api",
			"version": "0.1.0-minimal",
		})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Authentication endpoints
		auth := v1.Group("/auth")
		{
			auth.POST("/login", func(c *gin.Context) {
				var loginReq struct {
					Username string `json:"username" binding:"required"`
					Password string `json:"password" binding:"required"`
				}

				if err := c.ShouldBindJSON(&loginReq); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
					return
				}

				// Simple hardcoded authentication (for demo purposes)
				// Default credentials: admin/admin
				if loginReq.Username == "admin" && loginReq.Password == "admin" {
					// Mock JWT token (in real implementation, this would be properly signed)
					token := "mock-jwt-token-" + fmt.Sprintf("%d", time.Now().Unix())

					c.JSON(http.StatusOK, gin.H{
						"token": token,
						"user": gin.H{
							"id":       "user-1",
							"username": "admin",
							"email":    "admin@ovncp.local",
							"role":     "admin",
							"name":     "Administrator",
						},
						"expires_in": 3600, // 1 hour
					})
					return
				}

				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			})

			auth.GET("/me", func(c *gin.Context) {
				// Simple token validation (just check if Authorization header exists)
				authHeader := c.GetHeader("Authorization")
				if authHeader == "" || len(authHeader) < 7 {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "No valid token provided"})
					return
				}

				// Mock user info
				c.JSON(http.StatusOK, gin.H{
					"id":       "user-1",
					"username": "admin",
					"email":    "admin@ovncp.local",
					"role":     "admin",
					"name":     "Administrator",
				})
			})

			auth.POST("/logout", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
			})
		}

		// Switches endpoints (mock implementation)
		v1.GET("/switches", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"switches": []gin.H{
					{
						"id":    "sw1",
						"name":  "web-tier",
						"ports": 5,
					},
					{
						"id":    "sw2",
						"name":  "app-tier",
						"ports": 10,
					},
				},
				"total": 2,
			})
		})

		v1.GET("/switches/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{
				"id":         id,
				"name":       "web-tier",
				"ports":      5,
				"created_at": "2024-01-20T10:00:00Z",
			})
		})

		// Routers endpoints
		v1.GET("/routers", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"routers": []gin.H{
					{
						"id":   "lr1",
						"name": "edge-router",
						"type": "gateway",
					},
				},
				"total": 1,
			})
		})

		// Topology endpoint
		v1.GET("/topology", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"nodes": []gin.H{
					{"id": "sw1", "label": "web-tier", "type": "switch"},
					{"id": "sw2", "label": "app-tier", "type": "switch"},
					{"id": "lr1", "label": "edge-router", "type": "router"},
				},
				"edges": []gin.H{
					{"from": "sw1", "to": "lr1", "type": "connection"},
					{"from": "sw2", "to": "lr1", "type": "connection"},
				},
			})
		})

		// Multi-tenancy endpoints
		v1.GET("/tenants", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"tenants": []gin.H{
					{
						"id":           "tenant-123",
						"name":         "acme-corp",
						"display_name": "ACME Corporation",
						"type":         "organization",
						"status":       "active",
					},
				},
				"total": 1,
			})
		})
	}

	// API documentation
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OVN Control Platform API",
			"version": "0.1.0-minimal",
			"endpoints": gin.H{
				"health":   "/health",
				"api":      "/api/v1",
				"auth":     "/api/v1/auth/login",
				"switches": "/api/v1/switches",
				"routers":  "/api/v1/routers",
				"topology": "/api/v1/topology",
				"tenants":  "/api/v1/tenants",
			},
			"auth": gin.H{
				"default_credentials": gin.H{
					"username": "admin",
					"password": "admin",
				},
			},
		})
	})

	// Start server
	port := ":8080"
	fmt.Printf("Starting OVN Control Platform API on %s\n", port)
	fmt.Println("Endpoints:")
	fmt.Println("  http://localhost:8080/")
	fmt.Println("  http://localhost:8080/health")
	fmt.Println("  http://localhost:8080/api/v1/auth/login")
	fmt.Println("  http://localhost:8080/api/v1/switches")
	fmt.Println("  http://localhost:8080/api/v1/topology")
	fmt.Println("")
	fmt.Println("Default login credentials:")
	fmt.Println("  Username: admin")
	fmt.Println("  Password: admin")

	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
