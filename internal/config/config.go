package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	API         APIConfig
	OVN         OVNConfig
	Database    DatabaseConfig
	Auth        AuthConfig
	Security    SecurityConfig
	Log         LogConfig
	Environment string
}

type APIConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type OVNConfig struct {
	NorthboundDB   string
	SouthboundDB   string
	Timeout        time.Duration
	MaxRetries     int
	MaxConnections int
}

type DatabaseConfig struct {
	Type            string
	Host            string
	Port            string
	Name            string
	Database        string // Alias for Name
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type AuthConfig struct {
	Enabled           bool
	JWTSecret         string
	TokenExpiration   time.Duration
	RefreshExpiration time.Duration
	SessionExpiry     time.Duration
	Providers         map[string]OAuthProvider
}

type SecurityConfig struct {
	// Rate limiting
	RateLimitEnabled bool
	RateLimitRPS     int
	RateLimitBurst   int
	
	// CORS
	CORSAllowOrigins []string
	
	// Audit logging
	AuditEnabled bool
	
	// HTTPS enforcement
	ForceHTTPS bool
	
	// Security headers
	CSPEnabled bool
	HSTSEnabled bool
	HSTSMaxAge int
}

type OAuthProvider struct {
	Type         string // "oidc" or "oauth2"
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	// OIDC specific
	IssuerURL string
	// OAuth2 specific
	AuthURL     string
	TokenURL    string
	UserInfoURL string
}

type LogConfig struct {
	Level  string
	Format string
	Output string
}

func Load() (*Config, error) {
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		API: APIConfig{
			Port:         getEnv("API_PORT", "8080"),
			Host:         getEnv("API_HOST", "0.0.0.0"),
			ReadTimeout:  getDurationEnv("API_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("API_WRITE_TIMEOUT", 15*time.Second),
		},
		OVN: OVNConfig{
			NorthboundDB:   getEnv("OVN_NORTHBOUND_DB", "tcp:127.0.0.1:6641"),
			SouthboundDB:   getEnv("OVN_SOUTHBOUND_DB", "tcp:127.0.0.1:6642"),
			Timeout:        getDurationEnv("OVN_TIMEOUT", 30*time.Second),
			MaxRetries:     getIntEnv("OVN_MAX_RETRIES", 3),
			MaxConnections: getIntEnv("OVN_MAX_CONNECTIONS", 10),
		},
		Database: DatabaseConfig{
			Type:     getEnv("DB_TYPE", "postgres"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "ovncp"),
			User:     getEnv("DB_USER", "ovncp"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Auth: AuthConfig{
			Enabled:           getBoolEnv("AUTH_ENABLED", false),
			JWTSecret:         getEnv("JWT_SECRET", ""),
			TokenExpiration:   getDurationEnv("TOKEN_EXPIRATION", 24*time.Hour),
			RefreshExpiration: getDurationEnv("REFRESH_EXPIRATION", 7*24*time.Hour),
			SessionExpiry:     getDurationEnv("SESSION_EXPIRY", 7*24*time.Hour),
			Providers:         loadOAuthProviders(),
		},
		Security: SecurityConfig{
			RateLimitEnabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
			RateLimitRPS:     getIntEnv("RATE_LIMIT_RPS", 100),
			RateLimitBurst:   getIntEnv("RATE_LIMIT_BURST", 200),
			CORSAllowOrigins: getStringSliceEnv("CORS_ALLOW_ORIGINS", []string{"http://localhost:3000"}),
			AuditEnabled:     getBoolEnv("AUDIT_ENABLED", true),
			ForceHTTPS:       getBoolEnv("FORCE_HTTPS", false),
			CSPEnabled:       getBoolEnv("CSP_ENABLED", true),
			HSTSEnabled:      getBoolEnv("HSTS_ENABLED", true),
			HSTSMaxAge:       getIntEnv("HSTS_MAX_AGE", 31536000), // 1 year
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.Auth.Enabled && c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required when AUTH_ENABLED is true")
	}
	
	// OAuth providers are optional - we can use local auth
	// if c.Auth.Enabled && len(c.Auth.Providers) == 0 {
	// 	return fmt.Errorf("at least one OAuth provider must be configured when AUTH_ENABLED is true")
	// }
	
	for name, provider := range c.Auth.Providers {
		if provider.ClientID == "" || provider.ClientSecret == "" {
			return fmt.Errorf("OAuth provider %s is missing client credentials", name)
		}
		if provider.Type == "oidc" && provider.IssuerURL == "" {
			return fmt.Errorf("OIDC provider %s is missing issuer URL", name)
		}
		if provider.Type == "oauth2" && (provider.AuthURL == "" || provider.TokenURL == "") {
			return fmt.Errorf("OAuth2 provider %s is missing auth or token URL", name)
		}
	}
	
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	fmt.Sscanf(value, "%d", &result)
	return result
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func loadOAuthProviders() map[string]OAuthProvider {
	providers := make(map[string]OAuthProvider)
	
	// Load GitHub OAuth provider
	if getEnv("OAUTH_GITHUB_CLIENT_ID", "") != "" {
		providers["github"] = OAuthProvider{
			Type:         "oauth2",
			ClientID:     getEnv("OAUTH_GITHUB_CLIENT_ID", ""),
			ClientSecret: getEnv("OAUTH_GITHUB_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OAUTH_GITHUB_REDIRECT_URL", ""),
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			Scopes:       []string{"read:user", "user:email"},
		}
	}
	
	// Load Google OIDC provider
	if getEnv("OAUTH_GOOGLE_CLIENT_ID", "") != "" {
		providers["google"] = OAuthProvider{
			Type:         "oidc",
			ClientID:     getEnv("OAUTH_GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("OAUTH_GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OAUTH_GOOGLE_REDIRECT_URL", ""),
			IssuerURL:    "https://accounts.google.com",
			Scopes:       []string{"openid", "email", "profile"},
		}
	}
	
	// Load custom OIDC provider
	if getEnv("OAUTH_OIDC_CLIENT_ID", "") != "" {
		providers["oidc"] = OAuthProvider{
			Type:         "oidc",
			ClientID:     getEnv("OAUTH_OIDC_CLIENT_ID", ""),
			ClientSecret: getEnv("OAUTH_OIDC_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OAUTH_OIDC_REDIRECT_URL", ""),
			IssuerURL:    getEnv("OAUTH_OIDC_ISSUER_URL", ""),
			Scopes:       getStringSliceEnv("OAUTH_OIDC_SCOPES", []string{"openid", "email", "profile"}),
		}
	}
	
	return providers
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Simple comma-separated parsing
	var result []string
	for _, s := range splitString(value, ",") {
		if trimmed := trimString(s); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for start < end && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// GetBackupPath returns the backup storage path
func (c *Config) GetBackupPath() string {
	path := getEnv("BACKUP_PATH", "/var/lib/ovncp/backups")
	if !filepath.IsAbs(path) {
		// Make it absolute relative to current directory
		pwd, _ := os.Getwd()
		path = filepath.Join(pwd, path)
	}
	return path
}