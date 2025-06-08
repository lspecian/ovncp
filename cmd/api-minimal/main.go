package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

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
		// Switches endpoints (mock implementation)
		v1.GET("/switches", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"switches": []gin.H{
					{
						"id":   "sw1",
						"name": "web-tier",
						"ports": 5,
					},
					{
						"id":   "sw2",
						"name": "app-tier",
						"ports": 10,
					},
				},
				"total": 2,
			})
		})

		v1.GET("/switches/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{
				"id":          id,
				"name":        "web-tier",
				"ports":       5,
				"created_at":  "2024-01-20T10:00:00Z",
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
				"switches": "/api/v1/switches",
				"routers":  "/api/v1/routers",
				"topology": "/api/v1/topology",
				"tenants":  "/api/v1/tenants",
			},
		})
	})

	// Start server
	port := ":8080"
	fmt.Printf("Starting OVN Control Platform API on %s\n", port)
	fmt.Println("Endpoints:")
	fmt.Println("  http://localhost:8080/")
	fmt.Println("  http://localhost:8080/health")
	fmt.Println("  http://localhost:8080/api/v1/switches")
	fmt.Println("  http://localhost:8080/api/v1/topology")
	
	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}