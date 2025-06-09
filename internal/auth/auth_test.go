package auth

import (
	"testing"
	
	"github.com/lspecian/ovncp/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewProvider_OIDC(t *testing.T) {
	t.Skip("Skipping OIDC test - requires real OIDC endpoint")
	
	cfg := config.OAuthProvider{
		Type:         "oidc",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		IssuerURL:    "https://test.issuer.com",
		RedirectURL:  "https://app.example.com/callback",
		Scopes:       []string{"openid", "email", "profile"},
	}
	
	provider, err := NewProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_OAuth2(t *testing.T) {
	cfg := config.OAuthProvider{
		Type:         "oauth2",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://auth.example.com/auth",
		TokenURL:     "https://auth.example.com/token",
		UserInfoURL:  "https://auth.example.com/userinfo",
		RedirectURL:  "https://app.example.com/callback",
		Scopes:       []string{"read:user"},
	}
	
	provider, err := NewProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_InvalidType(t *testing.T) {
	cfg := config.OAuthProvider{
		Type: "invalid",
	}
	
	provider, err := NewProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "unsupported provider type")
}