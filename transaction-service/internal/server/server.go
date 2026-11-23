package server

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/InsafMin/gomoney/shared/proto"
	"github.com/InsafMin/gomoney/transaction-service/internal/handler"
)

type Server struct {
	port       int
	grpcServer *grpc.Server
	handler    *handler.Handler
}

func NewServer(port int, handler *handler.Handler) *Server {
	return &Server{
		port:    port,
		handler: handler,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterTransactionServiceServer(s.grpcServer, s.handler)
	reflection.Register(s.grpcServer)

	log.Printf("Starting gRPC server on port %d", s.port)

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		log.Println("Stopping gRPC server...")
		s.grpcServer.GracefulStop()
	}
}
