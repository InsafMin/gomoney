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

type TransactionClient struct {
	conn   *grpc.ClientConn
	client pb.TransactionServiceClient
	cb     *gobreaker.CircuitBreaker
}

func NewTransactionClient(serviceURL string) (*TransactionClient, error) {
	conn, err := grpc.NewClient(
		serviceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to transaction service: %w", err)
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "TransactionService",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	})

	log.Printf("Connected to Transaction Service at %s", serviceURL)

	return &TransactionClient{
		conn:   conn,
		client: pb.NewTransactionServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *TransactionClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *TransactionClient) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.CreateTransaction(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.CreateTransactionResponse), nil
}

func (c *TransactionClient) GetTransactions(ctx context.Context, req *pb.GetTransactionsRequest) (*pb.GetTransactionsResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetTransactions(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.GetTransactionsResponse), nil
}

func (c *TransactionClient) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetTransaction(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.GetTransactionResponse), nil
}

func (c *TransactionClient) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetBalance(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.GetBalanceResponse), nil
}

func (c *TransactionClient) GetCategories(ctx context.Context, req *pb.GetCategoriesRequest) (*pb.GetCategoriesResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetCategories(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.GetCategoriesResponse), nil
}

func (c *TransactionClient) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.CreateCategory(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.CreateCategoryResponse), nil
}
