package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/InsafMin/gomoney/auth-service/internal/jwt"
	"github.com/InsafMin/gomoney/auth-service/internal/models"
	"github.com/InsafMin/gomoney/auth-service/internal/password"
	"github.com/InsafMin/gomoney/auth-service/internal/repository"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Service интерфейс бизнес-логики
type Service interface {
	Register(ctx context.Context, email, pwd, firstName, lastName string) (*models.User, error)
	Login(ctx context.Context, email, pwd string) (*models.TokenPair, *models.UserProfile, error)
	RefreshToken(ctx context.Context, refreshToken string) (*models.TokenPair, error)
	ValidateToken(ctx context.Context, token string) (bool, string, string, error)
	GetProfile(ctx context.Context, userID string) (*models.UserProfile, error)
	UpdateProfile(ctx context.Context, userID, firstName, lastName, email string) (*models.UserProfile, error)
	DeleteUser(ctx context.Context, userID string) error
	CheckEmailAvailability(ctx context.Context, email string) (bool, error)
}

// AuthService реализация Service
type AuthService struct {
	repo           repository.Repository
	jwtManager     jwt.Manager
	passwordHasher password.Hasher
	redisClient    *redis.Client
	passwordMinLen int
}

// NewService создает новый AuthService
func NewService(
	repo repository.Repository,
	jwtManager jwt.Manager,
	passwordHasher password.Hasher,
	redisClient *redis.Client,
	passwordMinLen int,
) *AuthService {
	return &AuthService{
		repo:           repo,
		jwtManager:     jwtManager,
		passwordHasher: passwordHasher,
		redisClient:    redisClient,
		passwordMinLen: passwordMinLen,
	}
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(ctx context.Context, email, pwd, firstName, lastName string) (*models.User, error) {
	// Валидация email
	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("invalid email format")
	}

	// Проверка существования пользователя
	existingUser, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	// Валидация пароля
	if err := password.ValidatePasswordStrength(pwd, s.passwordMinLen); err != nil {
		return nil, fmt.Errorf("password validation failed: %w", err)
	}

	// Хеширование пароля
	hashedPassword, err := s.passwordHasher.Hash(pwd)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создание пользователя
	user := &models.User{
		Email:        email,
		PasswordHash: hashedPassword,
		FirstName:    firstName,
		LastName:     lastName,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("User registered successfully: %s (ID: %s)", email, user.ID)
	return user, nil
}

// Login выполняет вход пользователя
func (s *AuthService) Login(ctx context.Context, email, pwd string) (*models.TokenPair, *models.UserProfile, error) {
	// Получение пользователя
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	// Проверка активности
	if !user.IsActive {
		return nil, nil, fmt.Errorf("user account is inactive")
	}

	// Проверка пароля
	if err := s.passwordHasher.Compare(user.PasswordHash, pwd); err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	// Генерация токенов
	accessToken, accessExpiresAt, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExpiresAt, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохранение refresh токена в БД
	refreshTokenHash := hashToken(refreshToken)
	tokenModel := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: refreshExpiresAt,
	}

	if err := s.repo.CreateRefreshToken(ctx, tokenModel); err != nil {
		log.Printf("Warning: failed to save refresh token: %v", err)
	}

	// Обновление времени последнего входа
	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Printf("Warning: failed to update last login: %v", err)
	}

	// Кеширование информации о пользователе
	s.cacheUserProfile(ctx, user)

	tokenPair := &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
	}

	profile := &models.UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	log.Printf("User logged in successfully: %s (ID: %s)", email, user.ID)
	return tokenPair, profile, nil
}

// RefreshToken обновляет access токен
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	// Валидация refresh токена
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Проверка существования токена в БД
	tokenHash := hashToken(refreshToken)
	storedToken, err := s.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Проверка соответствия пользователя
	if storedToken.UserID != claims.UserID {
		return nil, fmt.Errorf("token user mismatch")
	}

	// Получение пользователя
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Генерация новых токенов
	newAccessToken, accessExpiresAt, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, refreshExpiresAt, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Отзыв старого токена
	if err := s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		log.Printf("Warning: failed to revoke old token: %v", err)
	}

	// Сохранение нового refresh токена
	newTokenHash := hashToken(newRefreshToken)
	tokenModel := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: refreshExpiresAt,
	}

	if err := s.repo.CreateRefreshToken(ctx, tokenModel); err != nil {
		log.Printf("Warning: failed to save new refresh token: %v", err)
	}

	tokenPair := &models.TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    accessExpiresAt,
	}

	log.Printf("Token refreshed for user ID: %s", user.ID)
	return tokenPair, nil
}

// ValidateToken проверяет валидность токена
func (s *AuthService) ValidateToken(ctx context.Context, token string) (bool, string, string, error) {
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return false, "", "", err
	}

	return true, claims.UserID, claims.Email, nil
}

// GetProfile получает профиль пользователя
func (s *AuthService) GetProfile(ctx context.Context, userID string) (*models.UserProfile, error) {
	// Попытка получить из кеша
	if profile := s.getCachedUserProfile(ctx, userID); profile != nil {
		return profile, nil
	}

	// Получение из БД
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	profile := &models.UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	// Кеширование
	s.cacheUserProfile(ctx, user)

	return profile, nil
}

// UpdateProfile обновляет профиль пользователя
func (s *AuthService) UpdateProfile(ctx context.Context, userID, firstName, lastName, email string) (*models.UserProfile, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Обновление полей если они указаны
	if firstName != "" {
		user.FirstName = firstName
	}
	if lastName != "" {
		user.LastName = lastName
	}
	if email != "" {
		if !emailRegex.MatchString(email) {
			return nil, fmt.Errorf("invalid email format")
		}
		user.Email = email
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидация кеша
	s.invalidateUserCache(ctx, userID)

	profile := &models.UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	log.Printf("Profile updated for user ID: %s", userID)
	return profile, nil
}

// DeleteUser удаляет пользователя
func (s *AuthService) DeleteUser(ctx context.Context, userID string) error {
	// Отзыв всех токенов
	if err := s.repo.RevokeAllUserTokens(ctx, userID); err != nil {
		log.Printf("Warning: failed to revoke user tokens: %v", err)
	}

	// Удаление пользователя
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Инвалидация кеша
	s.invalidateUserCache(ctx, userID)

	log.Printf("User deleted: ID %s", userID)
	return nil
}

// CheckEmailAvailability проверяет доступность email
func (s *AuthService) CheckEmailAvailability(ctx context.Context, email string) (bool, error) {
	if !emailRegex.MatchString(email) {
		return false, fmt.Errorf("invalid email format")
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// Email доступен
		return true, nil
	}

	if user != nil {
		// Email уже занят
		return false, nil
	}

	return true, nil
}

// Helper functions

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) cacheUserProfile(ctx context.Context, user *models.User) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("user:profile:%s", user.ID)
	data := fmt.Sprintf("%s:%s:%s", user.Email, user.FirstName, user.LastName)

	if err := s.redisClient.Set(ctx, key, data, 15*time.Minute).Err(); err != nil {
		log.Printf("Warning: failed to cache user profile: %v", err)
	}
}

func (s *AuthService) getCachedUserProfile(ctx context.Context, userID string) *models.UserProfile {
	if s.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("user:profile:%s", userID)
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	// Простой парсинг для примера (в реальности можно использовать JSON)
	_ = data
	return nil
}

func (s *AuthService) invalidateUserCache(ctx context.Context, userID string) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("user:profile:%s", userID)
	if err := s.redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("Warning: failed to invalidate user cache: %v", err)
	}
}
