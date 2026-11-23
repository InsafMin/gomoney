package handler

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/InsafMin/gomoney/auth-service/internal/service"
	pb "github.com/InsafMin/gomoney/shared/proto"
)

// Handler реализует gRPC методы AuthService
type Handler struct {
	pb.UnimplementedAuthServiceServer
	service service.Service
}

// NewHandler создает новый Handler
func NewHandler(service service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Register регистрирует нового пользователя
func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Register request: email=%s, first_name=%s, last_name=%s", req.Email, req.FirstName, req.LastName)

	user, err := h.service.Register(ctx, req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		log.Printf("Register error: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	profile := &pb.UserProfile{
		Id:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}

	return &pb.RegisterResponse{
		UserId:    user.ID,
		Message:   "User registered successfully",
		CreatedAt: user.CreatedAt.Unix(),
		Profile:   profile,
	}, nil
}

// Login выполняет вход пользователя
func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("Login request: email=%s", req.Email)

	tokenPair, profile, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		log.Printf("Login error: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	userProfile := &pb.UserProfile{
		Id:        profile.ID,
		Email:     profile.Email,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		CreatedAt: profile.CreatedAt.Unix(),
		UpdatedAt: profile.UpdatedAt.Unix(),
	}

	return &pb.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		Profile:      userProfile,
	}, nil
}

// RefreshToken обновляет access токен
func (h *Handler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	log.Printf("RefreshToken request")

	tokenPair, err := h.service.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		log.Printf("RefreshToken error: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token")
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
	}, nil
}

// ValidateToken проверяет валидность токена
func (h *Handler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	valid, userID, email, err := h.service.ValidateToken(ctx, req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:  false,
			UserId: "",
			Email:  "",
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:  valid,
		UserId: userID,
		Email:  email,
	}, nil
}

// GetProfile получает профиль пользователя
func (h *Handler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	log.Printf("GetProfile request: user_id=%s", req.UserId)

	profile, err := h.service.GetProfile(ctx, req.UserId)
	if err != nil {
		log.Printf("GetProfile error: %v", err)
		return &pb.GetProfileResponse{
			Success: false,
			Message: err.Error(),
			Profile: nil,
		}, nil
	}

	userProfile := &pb.UserProfile{
		Id:        profile.ID,
		Email:     profile.Email,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		CreatedAt: profile.CreatedAt.Unix(),
		UpdatedAt: profile.UpdatedAt.Unix(),
	}

	return &pb.GetProfileResponse{
		Success: true,
		Message: "Profile retrieved successfully",
		Profile: userProfile,
	}, nil
}

// UpdateProfile обновляет профиль пользователя
func (h *Handler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	log.Printf("UpdateProfile request: user_id=%s", req.UserId)

	firstName := ""
	lastName := ""
	email := ""

	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	if req.LastName != nil {
		lastName = *req.LastName
	}
	if req.Email != nil {
		email = *req.Email
	}

	profile, err := h.service.UpdateProfile(ctx, req.UserId, firstName, lastName, email)
	if err != nil {
		log.Printf("UpdateProfile error: %v", err)
		return &pb.UpdateProfileResponse{
			Success: false,
			Message: err.Error(),
			Profile: nil,
		}, nil
	}

	userProfile := &pb.UserProfile{
		Id:        profile.ID,
		Email:     profile.Email,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		CreatedAt: profile.CreatedAt.Unix(),
		UpdatedAt: profile.UpdatedAt.Unix(),
	}

	return &pb.UpdateProfileResponse{
		Success: true,
		Message: "Profile updated successfully",
		Profile: userProfile,
	}, nil
}

// DeleteUser удаляет пользователя
func (h *Handler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	log.Printf("DeleteUser request: user_id=%s", req.UserId)

	err := h.service.DeleteUser(ctx, req.UserId)
	if err != nil {
		log.Printf("DeleteUser error: %v", err)
		return &pb.DeleteUserResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

// CheckEmail проверяет доступность email
func (h *Handler) CheckEmail(ctx context.Context, req *pb.CheckEmailRequest) (*pb.CheckEmailResponse, error) {
	log.Printf("CheckEmail request: email=%s", req.Email)

	available, err := h.service.CheckEmailAvailability(ctx, req.Email)
	if err != nil {
		return &pb.CheckEmailResponse{
			Available: false,
			Message:   err.Error(),
		}, nil
	}

	message := "Email is available"
	if !available {
		message = "Email is already taken"
	}

	return &pb.CheckEmailResponse{
		Available: available,
		Message:   message,
	}, nil
}
