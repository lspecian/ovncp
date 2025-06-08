package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	
	"github.com/lspecian/ovncp/internal/config"
)

type UserInfo struct {
	ID      string
	Email   string
	Name    string
	Picture string
}

type Provider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
}

func NewProvider(cfg config.OAuthProvider) (Provider, error) {
	switch cfg.Type {
	case "oidc":
		return newOIDCProvider(cfg)
	case "oauth2":
		return newOAuth2Provider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
}

// OIDC Provider
type oidcProvider struct {
	config   *oauth2.Config
	provider *oidc.Provider
}

func newOIDCProvider(cfg config.OAuthProvider) (*oidcProvider, error) {
	ctx := context.Background()
	
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}
	
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}
	
	return &oidcProvider{
		config:   oauth2Config,
		provider: provider,
	}, nil
}

func (p *oidcProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *oidcProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *oidcProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	userInfo, err := p.provider.UserInfo(ctx, p.config.TokenSource(ctx, token))
	if err != nil {
		return nil, err
	}
	
	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	
	if err := userInfo.Claims(&claims); err != nil {
		return nil, err
	}
	
	return &UserInfo{
		ID:      claims.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
	}, nil
}

// OAuth2 Provider
type oauth2Provider struct {
	config      *oauth2.Config
	userInfoURL string
}

func newOAuth2Provider(cfg config.OAuthProvider) *oauth2Provider {
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.AuthURL,
			TokenURL: cfg.TokenURL,
		},
		Scopes: cfg.Scopes,
	}
	
	return &oauth2Provider{
		config:      oauth2Config,
		userInfoURL: cfg.UserInfoURL,
	}
}

func (p *oauth2Provider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *oauth2Provider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *oauth2Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)
	
	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}
	
	// GitHub-specific response structure
	// This can be made more generic by accepting different response formats
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	
	userInfo := &UserInfo{}
	
	// Extract ID
	if id, ok := data["id"]; ok {
		userInfo.ID = fmt.Sprintf("%v", id)
	}
	
	// Extract email
	if email, ok := data["email"].(string); ok {
		userInfo.Email = email
	}
	
	// Extract name
	if name, ok := data["name"].(string); ok {
		userInfo.Name = name
	} else if login, ok := data["login"].(string); ok {
		userInfo.Name = login
	}
	
	// Extract picture
	if picture, ok := data["avatar_url"].(string); ok {
		userInfo.Picture = picture
	} else if picture, ok := data["picture"].(string); ok {
		userInfo.Picture = picture
	}
	
	return userInfo, nil
}