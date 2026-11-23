package server

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/InsafMin/gomoney/auth-service/internal/handler"
	pb "github.com/InsafMin/gomoney/shared/proto"
)

// Server представляет gRPC сервер
type Server struct {
	port       int
	grpcServer *grpc.Server
	handler    *handler.Handler
}

// NewServer создает новый gRPC сервер
func NewServer(port int, handler *handler.Handler) *Server {
	return &Server{
		port:    port,
		handler: handler,
	}
}

// Start запускает gRPC сервер
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterAuthServiceServer(s.grpcServer, s.handler)

	// Регистрация reflection для отладки
	reflection.Register(s.grpcServer)

	log.Printf("Starting gRPC server on port %d", s.port)

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop останавливает gRPC сервер
func (s *Server) Stop() {
	if s.grpcServer != nil {
		log.Println("Stopping gRPC server...")
		s.grpcServer.GracefulStop()
	}
}
