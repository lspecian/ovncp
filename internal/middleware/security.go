package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	// CSP settings
	CSPEnabled          bool
	CSPDirectives       map[string][]string
	CSPReportOnly       bool
	CSPReportURI        string
	
	// HSTS settings
	HSTSEnabled         bool
	HSTSMaxAge          int
	HSTSIncludeSubdomains bool
	HSTSPreload         bool
	
	// Frame options
	XFrameOptions       string // DENY, SAMEORIGIN, or ALLOW-FROM uri
	
	// Content type options
	XContentTypeOptions string // nosniff
	
	// XSS Protection
	XXSSProtection      string // 1; mode=block
	
	// Referrer Policy
	ReferrerPolicy      string // no-referrer, same-origin, etc.
	
	// Permissions Policy
	PermissionsPolicy   map[string]string
	
	// CORS settings (if not using gin-contrib/cors)
	CORSEnabled         bool
	CORSAllowOrigins    []string
	CORSAllowMethods    []string
	CORSAllowHeaders    []string
	CORSExposeHeaders   []string
	CORSAllowCredentials bool
	CORSMaxAge          int
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		// Content Security Policy
		CSPEnabled: true,
		CSPDirectives: map[string][]string{
			"default-src": {"'self'"},
			"script-src":  {"'self'", "'unsafe-inline'", "'unsafe-eval'", "https://cdn.jsdelivr.net"},
			"style-src":   {"'self'", "'unsafe-inline'", "https://fonts.googleapis.com"},
			"font-src":    {"'self'", "https://fonts.gstatic.com"},
			"img-src":     {"'self'", "data:", "https:"},
			"connect-src": {"'self'", "wss:", "https:"},
			"frame-src":   {"'none'"},
			"object-src":  {"'none'"},
			"base-uri":    {"'self'"},
			"form-action": {"'self'"},
		},
		CSPReportOnly: false,
		CSPReportURI:  "/api/csp-report",
		
		// HTTP Strict Transport Security
		HSTSEnabled:           true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
		
		// Other security headers
		XFrameOptions:       "DENY",
		XContentTypeOptions: "nosniff",
		XXSSProtection:      "1; mode=block",
		ReferrerPolicy:      "strict-origin-when-cross-origin",
		
		// Permissions Policy
		PermissionsPolicy: map[string]string{
			"geolocation":        "()",
			"microphone":         "()",
			"camera":             "()",
			"payment":            "()",
			"usb":                "()",
			"magnetometer":       "()",
			"accelerometer":      "()",
			"gyroscope":          "()",
			"display-capture":    "()",
			"fullscreen":         "(self)",
			"picture-in-picture": "()",
		},
		
		// CORS defaults (restrictive)
		CORSEnabled:          true,
		CORSAllowOrigins:     []string{}, // Must be explicitly set
		CORSAllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		CORSExposeHeaders:    []string{"X-Request-ID"},
		CORSAllowCredentials: true,
		CORSMaxAge:           86400, // 24 hours
	}
}

// SecurityHeaders middleware adds security headers to responses
func SecurityHeaders(cfg SecurityConfig) gin.HandlerFunc {
	// Pre-build static header values
	hstsValue := buildHSTSHeader(cfg)
	permissionsPolicyValue := buildPermissionsPolicy(cfg.PermissionsPolicy)
	
	return func(c *gin.Context) {
		// Generate nonce for CSP
		nonce := ""
		if cfg.CSPEnabled {
			nonce = generateNonce()
			c.Set("csp-nonce", nonce)
		}
		
		// Add security headers
		headers := c.Writer.Header()
		
		// Content Security Policy
		if cfg.CSPEnabled {
			cspHeader := buildCSPHeader(cfg.CSPDirectives, nonce)
			if cfg.CSPReportOnly {
				headers.Set("Content-Security-Policy-Report-Only", cspHeader)
			} else {
				headers.Set("Content-Security-Policy", cspHeader)
			}
		}
		
		// HSTS
		if cfg.HSTSEnabled && hstsValue != "" {
			headers.Set("Strict-Transport-Security", hstsValue)
		}
		
		// Frame Options
		if cfg.XFrameOptions != "" {
			headers.Set("X-Frame-Options", cfg.XFrameOptions)
		}
		
		// Content Type Options
		if cfg.XContentTypeOptions != "" {
			headers.Set("X-Content-Type-Options", cfg.XContentTypeOptions)
		}
		
		// XSS Protection
		if cfg.XXSSProtection != "" {
			headers.Set("X-XSS-Protection", cfg.XXSSProtection)
		}
		
		// Referrer Policy
		if cfg.ReferrerPolicy != "" {
			headers.Set("Referrer-Policy", cfg.ReferrerPolicy)
		}
		
		// Permissions Policy
		if permissionsPolicyValue != "" {
			headers.Set("Permissions-Policy", permissionsPolicyValue)
		}
		
		// Remove potentially dangerous headers
		headers.Del("X-Powered-By")
		headers.Del("Server")
		
		c.Next()
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(cfg SecurityConfig) gin.HandlerFunc {
	if !cfg.CORSEnabled {
		return func(c *gin.Context) { c.Next() }
	}
	
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range cfg.CORSAllowOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}
		
		if allowed && origin != "" {
			// Set CORS headers
			c.Header("Access-Control-Allow-Origin", origin)
			
			if cfg.CORSAllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			
			// Handle preflight requests
			if c.Request.Method == "OPTIONS" {
				c.Header("Access-Control-Allow-Methods", strings.Join(cfg.CORSAllowMethods, ", "))
				c.Header("Access-Control-Allow-Headers", strings.Join(cfg.CORSAllowHeaders, ", "))
				
				if len(cfg.CORSExposeHeaders) > 0 {
					c.Header("Access-Control-Expose-Headers", strings.Join(cfg.CORSExposeHeaders, ", "))
				}
				
				if cfg.CORSMaxAge > 0 {
					c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.CORSMaxAge))
				}
				
				c.AbortWithStatus(204)
				return
			}
			
			// Set expose headers for actual requests
			if len(cfg.CORSExposeHeaders) > 0 {
				c.Header("Access-Control-Expose-Headers", strings.Join(cfg.CORSExposeHeaders, ", "))
			}
		}
		
		c.Next()
	}
}

// CSPReportHandler handles Content Security Policy violation reports
func CSPReportHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var report struct {
			CSPReport struct {
				DocumentURI        string `json:"document-uri"`
				Referrer           string `json:"referrer"`
				BlockedURI         string `json:"blocked-uri"`
				ViolatedDirective  string `json:"violated-directive"`
				EffectiveDirective string `json:"effective-directive"`
				OriginalPolicy     string `json:"original-policy"`
				Disposition        string `json:"disposition"`
				StatusCode         int    `json:"status-code"`
			} `json:"csp-report"`
		}
		
		if err := c.ShouldBindJSON(&report); err != nil {
			c.JSON(400, gin.H{"error": "Invalid CSP report"})
			return
		}
		
		// Log CSP violation
		// In production, you'd send this to your monitoring system
		fmt.Printf("CSP Violation: %+v\n", report.CSPReport)
		
		c.Status(204)
	}
}

// Helper functions

func generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

func buildCSPHeader(directives map[string][]string, nonce string) string {
	var parts []string
	
	for directive, values := range directives {
		// Add nonce to script-src if present
		if directive == "script-src" && nonce != "" {
			values = append(values, fmt.Sprintf("'nonce-%s'", nonce))
		}
		
		part := directive + " " + strings.Join(values, " ")
		parts = append(parts, part)
	}
	
	return strings.Join(parts, "; ")
}

func buildHSTSHeader(cfg SecurityConfig) string {
	if !cfg.HSTSEnabled {
		return ""
	}
	
	parts := []string{fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)}
	
	if cfg.HSTSIncludeSubdomains {
		parts = append(parts, "includeSubDomains")
	}
	
	if cfg.HSTSPreload {
		parts = append(parts, "preload")
	}
	
	return strings.Join(parts, "; ")
}

func buildPermissionsPolicy(policies map[string]string) string {
	var parts []string
	
	for feature, allowlist := range policies {
		parts = append(parts, fmt.Sprintf("%s=%s", feature, allowlist))
	}
	
	return strings.Join(parts, ", ")
}

// SecureRedirect middleware redirects HTTP to HTTPS
func SecureRedirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Header.Get("X-Forwarded-Proto") == "http" {
			url := "https://" + c.Request.Host + c.Request.URL.String()
			c.Redirect(301, url)
			c.Abort()
			return
		}
		c.Next()
	}
}