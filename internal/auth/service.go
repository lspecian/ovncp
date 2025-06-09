package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/models"
)

type Service interface {
	GetAuthURL(provider string, state string) (string, error)
	ExchangeCode(ctx context.Context, provider string, code string) (*models.Session, error)
	ValidateToken(ctx context.Context, token string) (*models.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*models.Session, error)
	Logout(ctx context.Context, token string) error
	GetUser(ctx context.Context, userID string) (*models.User, error)
	UpdateUserRole(ctx context.Context, userID string, role models.UserRole) error
	ListUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error)
	DeactivateUser(ctx context.Context, userID string) error
	LocalLogin(ctx context.Context, username, password string) (*models.Session, error)
}

type service struct {
	db        *sql.DB
	config    *config.AuthConfig
	providers map[string]Provider
}

func NewService(db *sql.DB, cfg *config.AuthConfig) (Service, error) {
	providers := make(map[string]Provider)
	
	for name, providerCfg := range cfg.Providers {
		provider, err := NewProvider(providerCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize provider %s: %w", name, err)
		}
		providers[name] = provider
	}
	
	return &service{
		db:        db,
		config:    cfg,
		providers: providers,
	}, nil
}

func (s *service) GetAuthURL(provider string, state string) (string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", fmt.Errorf("provider %s not found", provider)
	}
	
	return p.GetAuthURL(state), nil
}

func (s *service) ExchangeCode(ctx context.Context, provider string, code string) (*models.Session, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", provider)
	}
	
	// Exchange code for token
	token, err := p.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	
	// Get user info from provider
	userInfo, err := p.GetUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	
	// Find or create user
	user, err := s.findOrCreateUser(ctx, tx, provider, userInfo)
	if err != nil {
		return nil, err
	}
	
	// Update last login
	_, err = tx.ExecContext(ctx,
		"UPDATE users SET last_login_at = $1 WHERE id = $2",
		time.Now(), user.ID)
	if err != nil {
		return nil, err
	}
	
	// Create session
	session := &models.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		AccessToken:  generateToken(),
		RefreshToken: generateToken(),
		ExpiresAt:    time.Now().Add(s.config.TokenExpiration),
		CreatedAt:    time.Now(),
	}
	
	_, err = tx.ExecContext(ctx,
		"INSERT INTO sessions (id, user_id, access_token, refresh_token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		session.ID, session.UserID, session.AccessToken, session.RefreshToken, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return nil, err
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	
	// Attach user to session for response
	session.User = user
	
	return session, nil
}

func (s *service) findOrCreateUser(ctx context.Context, tx *sql.Tx, provider string, userInfo *UserInfo) (*models.User, error) {
	var user models.User
	
	// Try to find existing user
	err := tx.QueryRowContext(ctx,
		"SELECT id, email, name, picture, provider, provider_id, role, active, last_login_at, created_at, updated_at FROM users WHERE provider = $1 AND provider_id = $2",
		provider, userInfo.ID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Picture, &user.Provider,
		&user.ProviderID, &user.Role, &user.Active, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt)
	
	if err == sql.ErrNoRows {
		// Create new user
		user = models.User{
			ID:         uuid.New().String(),
			Email:      userInfo.Email,
			Name:       userInfo.Name,
			Picture:    userInfo.Picture,
			Provider:   provider,
			ProviderID: userInfo.ID,
			Role:       models.RoleViewer, // Default role
			Active:     true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		
		// Check if this is the first user - make them admin
		var count int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			user.Role = models.RoleAdmin
		}
		
		// Insert new user
		_, err = tx.ExecContext(ctx,
			"INSERT INTO users (id, email, name, picture, provider, provider_id, role, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
			user.ID, user.Email, user.Name, user.Picture, user.Provider,
			user.ProviderID, user.Role, user.Active, user.CreatedAt, user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		
		return &user, nil
	} else if err != nil {
		return nil, err
	}
	
	// Check if user is active
	if !user.Active {
		return nil, fmt.Errorf("user account is deactivated")
	}
	
	return &user, nil
}

func (s *service) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User
	
	err := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.email, u.name, u.picture, u.provider, u.provider_id, u.role, u.active, u.last_login_at, u.created_at, u.updated_at
		 FROM users u
		 INNER JOIN sessions s ON u.id = s.user_id
		 WHERE s.access_token = $1 AND s.expires_at > $2`,
		token, time.Now()).Scan(
		&user.ID, &user.Email, &user.Name, &user.Picture, &user.Provider,
		&user.ProviderID, &user.Role, &user.Active, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired token")
		}
		return nil, err
	}
	
	if !user.Active {
		return nil, fmt.Errorf("user account is deactivated")
	}
	
	return &user, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	
	// Find session by refresh token
	var oldSession models.Session
	err = tx.QueryRowContext(ctx,
		"SELECT id, user_id FROM sessions WHERE refresh_token = $1",
		refreshToken).Scan(&oldSession.ID, &oldSession.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, err
	}
	
	// Delete old session
	_, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE id = $1", oldSession.ID)
	if err != nil {
		return nil, err
	}
	
	// Create new session
	session := &models.Session{
		ID:           uuid.New().String(),
		UserID:       oldSession.UserID,
		AccessToken:  generateToken(),
		RefreshToken: generateToken(),
		ExpiresAt:    time.Now().Add(s.config.TokenExpiration),
		CreatedAt:    time.Now(),
	}
	
	_, err = tx.ExecContext(ctx,
		"INSERT INTO sessions (id, user_id, access_token, refresh_token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		session.ID, session.UserID, session.AccessToken, session.RefreshToken, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return nil, err
	}
	
	// Get user
	var user models.User
	err = tx.QueryRowContext(ctx,
		"SELECT id, email, name, picture, provider, provider_id, role, active, last_login_at, created_at, updated_at FROM users WHERE id = $1",
		session.UserID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Picture, &user.Provider,
		&user.ProviderID, &user.Role, &user.Active, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	
	if !user.Active {
		return nil, fmt.Errorf("user account is deactivated")
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	
	session.User = &user
	return session, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE access_token = $1",
		token)
	return err
}

func (s *service) GetUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	
	err := s.db.QueryRowContext(ctx,
		"SELECT id, email, name, picture, provider, provider_id, role, active, last_login_at, created_at, updated_at FROM users WHERE id = $1",
		userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Picture, &user.Provider,
		&user.ProviderID, &user.Role, &user.Active, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	
	return &user, nil
}

func (s *service) UpdateUserRole(ctx context.Context, userID string, role models.UserRole) error {
	result, err := s.db.ExecContext(ctx,
		"UPDATE users SET role = $1, updated_at = $2 WHERE id = $3",
		role, time.Now(), userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

func (s *service) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	
	// Get users
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, email, name, picture, provider, provider_id, role, active, last_login_at, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.Name, &user.Picture, &user.Provider,
			&user.ProviderID, &user.Role, &user.Active, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, &user)
	}
	
	return users, total, nil
}

func (s *service) DeactivateUser(ctx context.Context, userID string) error {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Deactivate user
	result, err := tx.ExecContext(ctx,
		"UPDATE users SET active = false, updated_at = $1 WHERE id = $2",
		time.Now(), userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	
	// Delete all user sessions
	_, err = tx.ExecContext(ctx,
		"DELETE FROM sessions WHERE user_id = $1",
		userID)
	if err != nil {
		return err
	}
	
	// Commit transaction
	return tx.Commit()
}

func generateToken() string {
	return uuid.New().String()
}

// LocalLogin authenticates a user with username and password
func (s *service) LocalLogin(ctx context.Context, username, password string) (*models.Session, error) {
	// For demo purposes, accept admin/admin credentials
	if username != "admin" || password != "admin" {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Find or create local admin user
	var user models.User
	err = tx.QueryRowContext(ctx,
		"SELECT id, email, name, picture, provider, provider_id, role, active, last_login_at, created_at, updated_at FROM users WHERE email = $1 AND provider = 'local'",
		"admin@ovncp.local").Scan(
		&user.ID, &user.Email, &user.Name, &user.Picture, &user.Provider,
		&user.ProviderID, &user.Role, &user.Active, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		// Create default admin user
		user = models.User{
			ID:         uuid.New().String(),
			Email:      "admin@ovncp.local",
			Name:       "Administrator",
			Picture:    "",
			Provider:   "local",
			ProviderID: "admin",
			Role:       models.RoleAdmin,
			Active:     true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		_, err = tx.ExecContext(ctx,
			"INSERT INTO users (id, email, name, picture, provider, provider_id, role, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
			user.ID, user.Email, user.Name, user.Picture, user.Provider,
			user.ProviderID, user.Role, user.Active, user.CreatedAt, user.UpdatedAt)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if !user.Active {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Update last login
	_, err = tx.ExecContext(ctx,
		"UPDATE users SET last_login_at = $1 WHERE id = $2",
		time.Now(), user.ID)
	if err != nil {
		return nil, err
	}

	// Create session
	session := &models.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		AccessToken:  generateToken(),
		RefreshToken: generateToken(),
		ExpiresAt:    time.Now().Add(s.config.TokenExpiration),
		CreatedAt:    time.Now(),
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO sessions (id, user_id, access_token, refresh_token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		session.ID, session.UserID, session.AccessToken, session.RefreshToken, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Attach user to session for response
	session.User = &user

	return session, nil
}