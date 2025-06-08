package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/metrics"
	"golang.org/x/time/rate"
)

// RateLimiter interface for different rate limiting strategies
type RateLimiter interface {
	Allow(key string) bool
	Limit() rate.Limit
	Burst() int
}

// IPRateLimiter implements per-IP rate limiting
type IPRateLimiter struct {
	ips    map[string]*rate.Limiter
	mu     sync.RWMutex
	limit  rate.Limit
	burst  int
	ttl    time.Duration
	lastGC time.Time
}

// NewIPRateLimiter creates a new IP-based rate limiter
func NewIPRateLimiter(rps float64, burst int, ttl time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		ips:    make(map[string]*rate.Limiter),
		limit:  rate.Limit(rps),
		burst:  burst,
		ttl:    ttl,
		lastGC: time.Now(),
	}
	
	// Start garbage collection goroutine
	go rl.gcLoop()
	
	return rl
}

// Allow checks if the request should be allowed
func (rl *IPRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	limiter, exists := rl.ips[key]
	if !exists {
		limiter = rate.NewLimiter(rl.limit, rl.burst)
		rl.ips[key] = limiter
	}
	
	return limiter.Allow()
}

// Limit returns the rate limit
func (rl *IPRateLimiter) Limit() rate.Limit {
	return rl.limit
}

// Burst returns the burst size
func (rl *IPRateLimiter) Burst() int {
	return rl.burst
}

// gcLoop periodically cleans up old entries
func (rl *IPRateLimiter) gcLoop() {
	ticker := time.NewTicker(rl.ttl)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.gc()
	}
}

// gc performs garbage collection
func (rl *IPRateLimiter) gc() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Simple GC: clear all entries periodically
	// In production, you'd track last access time
	if time.Since(rl.lastGC) > rl.ttl {
		rl.ips = make(map[string]*rate.Limiter)
		rl.lastGC = time.Now()
	}
}

// UserRateLimiter implements per-user rate limiting
type UserRateLimiter struct {
	users  map[string]*rate.Limiter
	mu     sync.RWMutex
	limits map[string]rate.Limit // Different limits per role
	burst  int
}

// NewUserRateLimiter creates a new user-based rate limiter
func NewUserRateLimiter(defaultLimit float64, burst int) *UserRateLimiter {
	return &UserRateLimiter{
		users: make(map[string]*rate.Limiter),
		limits: map[string]rate.Limit{
			"admin":   rate.Limit(1000), // 1000 req/s for admins
			"user":    rate.Limit(100),  // 100 req/s for users
			"api":     rate.Limit(500),  // 500 req/s for API keys
			"default": rate.Limit(defaultLimit),
		},
		burst: burst,
	}
}

// Allow checks if the request should be allowed for a user
func (rl *UserRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	limiter, exists := rl.users[key]
	if !exists {
		// Default limit for unknown users
		limiter = rate.NewLimiter(rl.limits["default"], rl.burst)
		rl.users[key] = limiter
	}
	
	return limiter.Allow()
}

// SetUserLimit sets a custom limit for a specific user
func (rl *UserRateLimiter) SetUserLimit(userID string, role string, customLimit *float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	var limit rate.Limit
	if customLimit != nil {
		limit = rate.Limit(*customLimit)
	} else if roleLimit, ok := rl.limits[role]; ok {
		limit = roleLimit
	} else {
		limit = rl.limits["default"]
	}
	
	rl.users[userID] = rate.NewLimiter(limit, rl.burst)
}

// Limit returns the default rate limit
func (rl *UserRateLimiter) Limit() rate.Limit {
	return rl.limits["default"]
}

// Burst returns the burst size
func (rl *UserRateLimiter) Burst() int {
	return rl.burst
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled       bool
	RequestsPerSecond float64
	Burst         int
	TTL           time.Duration
	ByUser        bool
	ByIP          bool
	CustomHeader  string // Custom header for rate limit key (e.g., "X-API-Key")
}

// RateLimit middleware factory
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}
	
	// Create rate limiters
	var ipLimiter *IPRateLimiter
	var userLimiter *UserRateLimiter
	
	if cfg.ByIP {
		ipLimiter = NewIPRateLimiter(cfg.RequestsPerSecond, cfg.Burst, cfg.TTL)
	}
	
	if cfg.ByUser {
		userLimiter = NewUserRateLimiter(cfg.RequestsPerSecond, cfg.Burst)
	}
	
	return func(c *gin.Context) {
		var allowed bool
		var limiter RateLimiter
		var key string
		
		// Check custom header first
		if cfg.CustomHeader != "" {
			key = c.GetHeader(cfg.CustomHeader)
			if key != "" && ipLimiter != nil {
				allowed = ipLimiter.Allow(key)
				limiter = ipLimiter
			}
		}
		
		// Check user-based rate limiting
		if cfg.ByUser && userLimiter != nil {
			if user, exists := c.Get("user"); exists {
				if u, ok := user.(map[string]interface{}); ok {
					if userID, ok := u["id"].(string); ok {
						key = "user:" + userID
						allowed = userLimiter.Allow(userID)
						limiter = userLimiter
						
						// Set user-specific limit based on role
						if roles, ok := u["roles"].([]string); ok && len(roles) > 0 {
							userLimiter.SetUserLimit(userID, roles[0], nil)
						}
					}
				}
			}
		}
		
		// Fall back to IP-based rate limiting
		if !allowed && cfg.ByIP && ipLimiter != nil {
			key = "ip:" + c.ClientIP()
			allowed = ipLimiter.Allow(c.ClientIP())
			limiter = ipLimiter
		}
		
		// If no rate limiter is configured, allow the request
		if limiter == nil {
			c.Next()
			return
		}
		
		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(int(limiter.Limit())))
		c.Header("X-RateLimit-Burst", strconv.Itoa(limiter.Burst()))
		
		if !allowed {
			// Record rate limit metric
			metrics.HTTPRequestsTotal.WithLabelValues(
				c.Request.Method,
				c.FullPath(),
				"429",
			).Inc()
			
			c.Header("X-RateLimit-Retry-After", "1")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests from %s", key),
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// EndpointRateLimit provides per-endpoint rate limiting
func EndpointRateLimit(rps float64, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%.0f", rps))
			c.Header("X-RateLimit-Burst", strconv.Itoa(burst))
			c.Header("X-RateLimit-Retry-After", "1")
			
			metrics.HTTPRequestsTotal.WithLabelValues(
				c.Request.Method,
				c.FullPath(),
				"429",
			).Inc()
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"message": "Endpoint rate limit exceeded",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// AdaptiveRateLimit implements adaptive rate limiting based on system load
type AdaptiveRateLimiter struct {
	base     rate.Limit
	current  rate.Limit
	burst    int
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	
	// Metrics for adaptation
	cpuThreshold    float64
	memoryThreshold uint64
	errorThreshold  float64
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(baseRPS float64, burst int) *AdaptiveRateLimiter {
	arl := &AdaptiveRateLimiter{
		base:            rate.Limit(baseRPS),
		current:         rate.Limit(baseRPS),
		burst:           burst,
		limiters:        make(map[string]*rate.Limiter),
		cpuThreshold:    0.8,  // 80% CPU
		memoryThreshold: 1<<30, // 1GB memory
		errorThreshold:  0.05,  // 5% error rate
	}
	
	// Start monitoring goroutine
	go arl.monitor()
	
	return arl
}

// monitor adjusts rate limits based on system metrics
func (arl *AdaptiveRateLimiter) monitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		// In a real implementation, you would check actual system metrics
		// For now, we'll use a simple adjustment logic
		
		arl.mu.Lock()
		// Adjust rate limit based on conditions
		// This is a simplified example
		if arl.current < arl.base {
			// Gradually increase back to base rate
			arl.current = rate.Limit(float64(arl.current) * 1.1)
			if arl.current > arl.base {
				arl.current = arl.base
			}
		}
		
		// Update all existing limiters
		for _, limiter := range arl.limiters {
			limiter.SetLimit(arl.current)
		}
		arl.mu.Unlock()
	}
}

// Allow checks if the request should be allowed
func (arl *AdaptiveRateLimiter) Allow(key string) bool {
	arl.mu.Lock()
	defer arl.mu.Unlock()
	
	limiter, exists := arl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(arl.current, arl.burst)
		arl.limiters[key] = limiter
	}
	
	return limiter.Allow()
}

// ReduceLimit reduces the current rate limit by a factor
func (arl *AdaptiveRateLimiter) ReduceLimit(factor float64) {
	arl.mu.Lock()
	defer arl.mu.Unlock()
	
	arl.current = rate.Limit(float64(arl.current) * factor)
	if arl.current < 1 {
		arl.current = 1 // Minimum 1 req/s
	}
}

// Limit returns the current rate limit
func (arl *AdaptiveRateLimiter) Limit() rate.Limit {
	arl.mu.RLock()
	defer arl.mu.RUnlock()
	return arl.current
}

// Burst returns the burst size
func (arl *AdaptiveRateLimiter) Burst() int {
	return arl.burst
}