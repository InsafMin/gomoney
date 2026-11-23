package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost - стандартная сложность хеширования bcrypt
	DefaultCost = 10
)

// Hasher интерфейс для хеширования паролей
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

// BcryptHasher реализация Hasher с использованием bcrypt
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher создает новый BcryptHasher
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: DefaultCost}
}

// Hash хеширует пароль
func (h *BcryptHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// Compare сравнивает хешированный пароль с обычным
func (h *BcryptHasher) Compare(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return fmt.Errorf("invalid password")
		}
		return fmt.Errorf("failed to compare passwords: %w", err)
	}
	return nil
}

// ValidatePasswordStrength проверяет соответствие пароля требованиям безопасности
func ValidatePasswordStrength(password string, minLength int) error {
	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters long", minLength)
	}
	if len(password) > 72 {
		return fmt.Errorf("password must not exceed 72 characters")
	}
	return nil
}
