package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/InsafMin/gomoney/auth-service/internal/jwt"
	"github.com/InsafMin/gomoney/auth-service/internal/models"
	"github.com/InsafMin/gomoney/auth-service/internal/password"
)

// Mock Repository
type mockRepository struct {
	users         map[string]*models.User
	usersByEmail  map[string]*models.User
	refreshTokens map[string]*models.RefreshToken
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:         make(map[string]*models.User),
		usersByEmail:  make(map[string]*models.User),
		refreshTokens: make(map[string]*models.RefreshToken),
	}
}

func (m *mockRepository) CreateUser(ctx context.Context, user *models.User) error {
	if _, exists := m.usersByEmail[user.Email]; exists {
		return errors.New("user already exists")
	}
	user.ID = "mock-user-id"
	user.IsActive = true
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *mockRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, exists := m.usersByEmail[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockRepository) UpdateUser(ctx context.Context, user *models.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return errors.New("user not found")
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *mockRepository) DeleteUser(ctx context.Context, id string) error {
	user, exists := m.users[id]
	if !exists {
		return errors.New("user not found")
	}
	delete(m.usersByEmail, user.Email)
	delete(m.users, id)
	return nil
}

func (m *mockRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	return nil
}

func (m *mockRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	m.refreshTokens[token.TokenHash] = token
	return nil
}

func (m *mockRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token, exists := m.refreshTokens[tokenHash]
	if !exists {
		return nil, errors.New("token not found")
	}
	if token.Revoked || token.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired or revoked")
	}
	return token, nil
}

func (m *mockRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	token, exists := m.refreshTokens[tokenHash]
	if exists {
		token.Revoked = true
		now := time.Now()
		token.RevokedAt = &now
	}
	return nil
}

func (m *mockRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	for _, token := range m.refreshTokens {
		if token.UserID == userID {
			token.Revoked = true
			now := time.Now()
			token.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRepository) CleanupExpiredTokens(ctx context.Context) error {
	return nil
}

// Tests

func TestAuthService_Register(t *testing.T) {
	repo := newMockRepository()
	jwtMgr := jwt.NewJWTManager("test-secret", 15*time.Minute, 168*time.Hour)
	pwdHasher := password.NewBcryptHasher()

	svc := NewService(repo, jwtMgr, pwdHasher, nil, 6)

	tests := []struct {
		name      string
		email     string
		password  string
		firstName string
		lastName  string
		wantError bool
	}{
		{
			name:      "successful registration",
			email:     "test@example.com",
			password:  "password123",
			firstName: "John",
			lastName:  "Doe",
			wantError: false,
		},
		{
			name:      "invalid email",
			email:     "invalid-email",
			password:  "password123",
			firstName: "John",
			lastName:  "Doe",
			wantError: true,
		},
		{
			name:      "weak password",
			email:     "test2@example.com",
			password:  "123",
			firstName: "Jane",
			lastName:  "Doe",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			user, err := svc.Register(ctx, tt.email, tt.password, tt.firstName, tt.lastName)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if user.Email != tt.email {
				t.Errorf("expected email %s, got %s", tt.email, user.Email)
			}

			if user.FirstName != tt.firstName {
				t.Errorf("expected first name %s, got %s", tt.firstName, user.FirstName)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	repo := newMockRepository()
	jwtMgr := jwt.NewJWTManager("test-secret", 15*time.Minute, 168*time.Hour)
	pwdHasher := password.NewBcryptHasher()

	svc := NewService(repo, jwtMgr, pwdHasher, nil, 6)

	ctx := context.Background()

	// Сначала зарегистрируем пользователя
	email := "login@example.com"
	password := "password123"
	_, err := svc.Register(ctx, email, password, "Test", "User")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	tests := []struct {
		name      string
		email     string
		password  string
		wantError bool
	}{
		{
			name:      "successful login",
			email:     email,
			password:  password,
			wantError: false,
		},
		{
			name:      "wrong password",
			email:     email,
			password:  "wrongpassword",
			wantError: true,
		},
		{
			name:      "non-existent user",
			email:     "nonexistent@example.com",
			password:  password,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPair, profile, err := svc.Login(ctx, tt.email, tt.password)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tokenPair.AccessToken == "" {
				t.Error("access token should not be empty")
			}

			if tokenPair.RefreshToken == "" {
				t.Error("refresh token should not be empty")
			}

			if profile.Email != tt.email {
				t.Errorf("expected email %s, got %s", tt.email, profile.Email)
			}
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	jwtMgr := jwt.NewJWTManager("test-secret", 15*time.Minute, 168*time.Hour)
	svc := NewService(nil, jwtMgr, nil, nil, 6)

	ctx := context.Background()

	// Генерируем валидный токен
	validToken, _, err := jwtMgr.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantValid bool
		wantError bool
	}{
		{
			name:      "valid token",
			token:     validToken,
			wantValid: true,
			wantError: false,
		},
		{
			name:      "invalid token",
			token:     "invalid.token.here",
			wantValid: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, userID, email, err := svc.ValidateToken(ctx, tt.token)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
				return
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if valid != tt.wantValid {
				t.Errorf("expected valid=%v, got %v", tt.wantValid, valid)
			}

			if tt.wantValid {
				if userID == "" {
					t.Error("userID should not be empty")
				}
				if email == "" {
					t.Error("email should not be empty")
				}
			}
		})
	}
}

func TestAuthService_CheckEmailAvailability(t *testing.T) {
	repo := newMockRepository()
	jwtMgr := jwt.NewJWTManager("test-secret", 15*time.Minute, 168*time.Hour)
	pwdHasher := password.NewBcryptHasher()

	svc := NewService(repo, jwtMgr, pwdHasher, nil, 6)

	ctx := context.Background()

	// Регистрируем пользователя
	_, err := svc.Register(ctx, "taken@example.com", "password123", "Test", "User")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	tests := []struct {
		name          string
		email         string
		wantAvailable bool
		wantError     bool
	}{
		{
			name:          "available email",
			email:         "available@example.com",
			wantAvailable: true,
			wantError:     false,
		},
		{
			name:          "taken email",
			email:         "taken@example.com",
			wantAvailable: false,
			wantError:     false,
		},
		{
			name:          "invalid email",
			email:         "invalid-email",
			wantAvailable: false,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available, err := svc.CheckEmailAvailability(ctx, tt.email)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
				return
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if available != tt.wantAvailable {
				t.Errorf("expected available=%v, got %v", tt.wantAvailable, available)
			}
		})
	}
}
