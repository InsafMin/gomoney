package clients

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/InsafMin/gomoney/shared/proto"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	client pb.AuthServiceClient
	cb     *gobreaker.CircuitBreaker
}

func NewAuthClient(serviceURL string) (*AuthClient, error) {
	conn, err := grpc.NewClient(
		serviceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "AuthService",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	})

	log.Printf("Connected to Auth Service at %s", serviceURL)

	return &AuthClient{
		conn:   conn,
		client: pb.NewAuthServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *AuthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *AuthClient) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.Register(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.RegisterResponse), nil
}

func (c *AuthClient) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.Login(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.LoginResponse), nil
}

func (c *AuthClient) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.ValidateToken(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.ValidateTokenResponse), nil
}

func (c *AuthClient) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.RefreshToken(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.RefreshTokenResponse), nil
}

func (c *AuthClient) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetProfile(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.GetProfileResponse), nil
}
