package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Name         string    `json:"name" db:"name"`
	Picture      string    `json:"picture,omitempty" db:"picture"`
	Provider     string    `json:"provider" db:"provider"`
	ProviderID   string    `json:"provider_id" db:"provider_id"`
	Role         UserRole  `json:"role" db:"role"`
	Active       bool      `json:"active" db:"active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserRole represents the role of a user
type UserRole string

const (
	// RoleAdmin has full access to all resources
	RoleAdmin UserRole = "admin"
	// RoleOperator can create, update, and delete resources
	RoleOperator UserRole = "operator"
	// RoleViewer can only read resources
	RoleViewer UserRole = "viewer"
)

// String returns the string representation of the role
func (r UserRole) String() string {
	return string(r)
}

// IsValid checks if the role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	}
	return false
}

// CanWrite checks if the role has write permissions
func (r UserRole) CanWrite() bool {
	return r == RoleAdmin || r == RoleOperator
}

// CanDelete checks if the role has delete permissions
func (r UserRole) CanDelete() bool {
	return r == RoleAdmin || r == RoleOperator
}

// CanManageUsers checks if the role can manage users
func (r UserRole) CanManageUsers() bool {
	return r == RoleAdmin
}

// Scan implements the sql.Scanner interface
func (r *UserRole) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("cannot scan nil into UserRole")
	}
	
	switch v := value.(type) {
	case string:
		*r = UserRole(v)
		if !r.IsValid() {
			return fmt.Errorf("invalid role: %s", v)
		}
		return nil
	case []byte:
		*r = UserRole(string(v))
		if !r.IsValid() {
			return fmt.Errorf("invalid role: %s", string(v))
		}
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into UserRole", value)
	}
}

// Value implements the driver.Valuer interface
func (r UserRole) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid role: %s", r)
	}
	return string(r), nil
}

// Session represents a user session
type Session struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	AccessToken  string    `json:"access_token" db:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty" db:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	User         *User     `json:"user,omitempty" db:"-"` // Not stored in DB
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// UserCreateRequest represents a request to create a user
type UserCreateRequest struct {
	Email string   `json:"email" binding:"required,email"`
	Name  string   `json:"name" binding:"required"`
	Role  UserRole `json:"role" binding:"required"`
}

// UserUpdateRequest represents a request to update a user
type UserUpdateRequest struct {
	Name   string   `json:"name,omitempty"`
	Role   UserRole `json:"role,omitempty"`
	Active *bool    `json:"active,omitempty"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         *User  `json:"user"`
}

// OAuthConfig represents OAuth provider configuration
type OAuthConfig struct {
	Provider     string   `json:"provider"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"-"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	AuthURL      string   `json:"auth_url,omitempty"`
	TokenURL     string   `json:"token_url,omitempty"`
	UserInfoURL  string   `json:"user_info_url,omitempty"`
}

// UserClaims represents the claims in the JWT token
type UserClaims struct {
	UserID   string   `json:"sub"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Role     UserRole `json:"role"`
	Provider string   `json:"provider"`
}