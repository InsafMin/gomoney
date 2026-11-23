package server

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/InsafMin/gomoney/analytics-service/internal/handler"
	pb "github.com/InsafMin/gomoney/shared/proto"
)

// Server представляет gRPC сервер аналитики
type Server struct {
	grpcServer *grpc.Server
	port       int
}

// NewServer создает новый gRPC сервер
func NewServer(port int, handler *handler.Handler) *Server {
	grpcServer := grpc.NewServer()

	pb.RegisterAnalyticsServiceServer(grpcServer, handler)

	// Регистрация health check сервиса
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Регистрация reflection для удобства отладки
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

// Start запускает gRPC сервер
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Printf("Analytics gRPC server starting on port %d", s.port)

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop останавливает gRPC сервер
func (s *Server) Stop() {
	log.Println("Stopping analytics gRPC server...")
	s.grpcServer.GracefulStop()
}
