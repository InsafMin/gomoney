package jwt

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", 15*time.Minute, 168*time.Hour)

	userID := "user-123"
	email := "test@example.com"

	token, expiresAt, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	if token == "" {
		t.Error("token should not be empty")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("expiration time should be in the future")
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", 15*time.Minute, 168*time.Hour)

	userID := "user-123"
	email := "test@example.com"

	token, _, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "valid token",
			token:     token,
			wantError: false,
		},
		{
			name:      "invalid token",
			token:     "invalid.token.here",
			wantError: true,
		},
		{
			name:      "empty token",
			token:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)

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

			if claims.UserID != userID {
				t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
			}

			if claims.Email != email {
				t.Errorf("expected Email %s, got %s", email, claims.Email)
			}
		})
	}
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", 15*time.Minute, 168*time.Hour)

	userID := "user-123"

	token, expiresAt, err := manager.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Error("refresh token should not be empty")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("expiration time should be in the future")
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", -1*time.Second, 168*time.Hour)

	userID := "user-123"
	email := "test@example.com"

	token, _, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}
