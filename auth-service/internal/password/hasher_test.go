package password

import (
	"strings"
	"testing"
)

func TestBcryptHasher_Hash(t *testing.T) {
	hasher := NewBcryptHasher()

	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{
			name:      "valid password",
			password:  "mySecurePassword123",
			wantError: false,
		},
		{
			name:      "empty password",
			password:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)

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

			if hash == "" {
				t.Error("hash should not be empty")
			}

			if hash == tt.password {
				t.Error("hash should not equal plain password")
			}
		})
	}
}

func TestBcryptHasher_Compare(t *testing.T) {
	hasher := NewBcryptHasher()
	password := "testPassword123"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	tests := []struct {
		name          string
		hashedPwd     string
		plainPwd      string
		expectError   bool
		errorContains string
	}{
		{
			name:        "correct password",
			hashedPwd:   hash,
			plainPwd:    password,
			expectError: false,
		},
		{
			name:          "incorrect password",
			hashedPwd:     hash,
			plainPwd:      "wrongPassword",
			expectError:   true,
			errorContains: "invalid password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.Compare(tt.hashedPwd, tt.plainPwd)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		minLength int
		wantError bool
	}{
		{
			name:      "valid password",
			password:  "password123",
			minLength: 6,
			wantError: false,
		},
		{
			name:      "too short",
			password:  "pass",
			minLength: 6,
			wantError: true,
		},
		{
			name:      "exactly min length",
			password:  "pass12",
			minLength: 6,
			wantError: false,
		},
		{
			name:      "too long (over 72 chars)",
			password:  strings.Repeat("a", 73),
			minLength: 6,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password, tt.minLength)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
