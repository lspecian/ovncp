package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
	
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/models"
)

// Mock Provider
type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) GetAuthURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *mockProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oauth2.Token), args.Error(1)
}

func (m *mockProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserInfo), args.Error(1)
}

func TestGetAuthURL(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	
	mockProv := new(mockProvider)
	mockProv.On("GetAuthURL", "test-state").Return("https://provider.com/auth?state=test-state")
	
	svc := &service{
		db:     db,
		config: &config.AuthConfig{},
		providers: map[string]Provider{
			"test": mockProv,
		},
	}
	
	url, err := svc.GetAuthURL("test", "test-state")
	assert.NoError(t, err)
	assert.Equal(t, "https://provider.com/auth?state=test-state", url)
	
	// Test non-existent provider
	_, err = svc.GetAuthURL("nonexistent", "test-state")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider nonexistent not found")
	
	mockProv.AssertExpectations(t)
}

func TestExchangeCode(t *testing.T) {
	db, dbMock, _ := sqlmock.New()
	defer db.Close()
	
	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "provider-token",
	}
	userInfo := &UserInfo{
		ID:      "123",
		Email:   "test@example.com",
		Name:    "Test User",
		Picture: "https://example.com/pic.jpg",
	}
	
	mockProv := new(mockProvider)
	mockProv.On("ExchangeCode", ctx, "test-code").Return(token, nil)
	mockProv.On("GetUserInfo", ctx, token).Return(userInfo, nil)
	
	svc := &service{
		db: db,
		config: &config.AuthConfig{
			TokenExpiration: 24 * time.Hour,
		},
		providers: map[string]Provider{
			"test": mockProv,
		},
	}
	
	// Test new user creation (first user = admin)
	dbMock.ExpectBegin()
	
	// Check for existing user
	dbMock.ExpectQuery("SELECT .+ FROM users WHERE provider = \\$1 AND provider_id = \\$2").
		WithArgs("test", "123").
		WillReturnError(sql.ErrNoRows)
	
	// Check user count for first admin
	dbMock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	
	// Insert new user
	dbMock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), "test@example.com", "Test User", "https://example.com/pic.jpg",
			"test", "123", "admin", true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	// Update last login
	dbMock.ExpectExec("UPDATE users SET last_login_at = \\$1 WHERE id = \\$2").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	// Insert session
	dbMock.ExpectExec("INSERT INTO sessions").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	dbMock.ExpectCommit()
	
	session, err := svc.ExchangeCode(ctx, "test", "test-code")
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.AccessToken)
	assert.NotEmpty(t, session.RefreshToken)
	assert.NotNil(t, session.User)
	assert.Equal(t, "test@example.com", session.User.Email)
	assert.Equal(t, models.RoleAdmin, session.User.Role) // First user is admin
	
	assert.NoError(t, dbMock.ExpectationsWereMet())
	mockProv.AssertExpectations(t)
}

func TestValidateToken(t *testing.T) {
	db, dbMock, _ := sqlmock.New()
	defer db.Close()
	
	ctx := context.Background()
	svc := &service{
		db:     db,
		config: &config.AuthConfig{},
	}
	
	// Test valid token
	dbMock.ExpectQuery("SELECT .+ FROM users u INNER JOIN sessions s").
		WithArgs("valid-token", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "name", "picture", "provider", "provider_id",
			"role", "active", "last_login_at", "created_at", "updated_at",
		}).AddRow(
			"user-123", "test@example.com", "Test User", "pic.jpg", "test", "123",
			"admin", true, time.Now(), time.Now(), time.Now(),
		))
	
	user, err := svc.ValidateToken(ctx, "valid-token")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, models.RoleAdmin, user.Role)
	
	// Test invalid token
	dbMock.ExpectQuery("SELECT .+ FROM users u INNER JOIN sessions s").
		WithArgs("invalid-token", sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	
	_, err = svc.ValidateToken(ctx, "invalid-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
	
	// Test deactivated user
	dbMock.ExpectQuery("SELECT .+ FROM users u INNER JOIN sessions s").
		WithArgs("deactivated-token", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "name", "picture", "provider", "provider_id",
			"role", "active", "last_login_at", "created_at", "updated_at",
		}).AddRow(
			"user-123", "test@example.com", "Test User", "pic.jpg", "test", "123",
			"admin", false, time.Now(), time.Now(), time.Now(),
		))
	
	_, err = svc.ValidateToken(ctx, "deactivated-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user account is deactivated")
	
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUpdateUserRole(t *testing.T) {
	db, dbMock, _ := sqlmock.New()
	defer db.Close()
	
	ctx := context.Background()
	svc := &service{
		db:     db,
		config: &config.AuthConfig{},
	}
	
	// Test successful update
	dbMock.ExpectExec("UPDATE users SET role = \\$1, updated_at = \\$2 WHERE id = \\$3").
		WithArgs("operator", sqlmock.AnyArg(), "user-123").
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	err := svc.UpdateUserRole(ctx, "user-123", models.RoleOperator)
	assert.NoError(t, err)
	
	// Test user not found
	dbMock.ExpectExec("UPDATE users SET role = \\$1, updated_at = \\$2 WHERE id = \\$3").
		WithArgs("admin", sqlmock.AnyArg(), "nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))
	
	err = svc.UpdateUserRole(ctx, "nonexistent", models.RoleAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
	
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestListUsers(t *testing.T) {
	db, dbMock, _ := sqlmock.New()
	defer db.Close()
	
	ctx := context.Background()
	svc := &service{
		db:     db,
		config: &config.AuthConfig{},
	}
	
	// Test successful list
	dbMock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	
	dbMock.ExpectQuery("SELECT .+ FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "name", "picture", "provider", "provider_id",
			"role", "active", "last_login_at", "created_at", "updated_at",
		}).AddRow(
			"user-1", "admin@example.com", "Admin", "pic1.jpg", "test", "1",
			"admin", true, time.Now(), time.Now(), time.Now(),
		).AddRow(
			"user-2", "viewer@example.com", "Viewer", "pic2.jpg", "test", "2",
			"viewer", true, time.Now(), time.Now(), time.Now(),
		))
	
	users, total, err := svc.ListUsers(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, users, 2)
	assert.Equal(t, "admin@example.com", users[0].Email)
	assert.Equal(t, models.RoleAdmin, users[0].Role)
	assert.Equal(t, "viewer@example.com", users[1].Email)
	assert.Equal(t, models.RoleViewer, users[1].Role)
	
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDeactivateUser(t *testing.T) {
	db, dbMock, _ := sqlmock.New()
	defer db.Close()
	
	ctx := context.Background()
	svc := &service{
		db:     db,
		config: &config.AuthConfig{},
	}
	
	// Test successful deactivation
	dbMock.ExpectBegin()
	
	dbMock.ExpectExec("UPDATE users SET active = false, updated_at = \\$1 WHERE id = \\$2").
		WithArgs(sqlmock.AnyArg(), "user-123").
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	dbMock.ExpectExec("DELETE FROM sessions WHERE user_id = \\$1").
		WithArgs("user-123").
		WillReturnResult(sqlmock.NewResult(1, 3)) // Deleted 3 sessions
	
	dbMock.ExpectCommit()
	
	err := svc.DeactivateUser(ctx, "user-123")
	assert.NoError(t, err)
	
	// Test user not found
	dbMock.ExpectBegin()
	
	dbMock.ExpectExec("UPDATE users SET active = false, updated_at = \\$1 WHERE id = \\$2").
		WithArgs(sqlmock.AnyArg(), "nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))
	
	dbMock.ExpectRollback()
	
	err = svc.DeactivateUser(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
	
	assert.NoError(t, dbMock.ExpectationsWereMet())
}