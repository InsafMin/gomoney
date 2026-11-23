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

type AnalyticsClient struct {
	conn   *grpc.ClientConn
	client pb.AnalyticsServiceClient
	cb     *gobreaker.CircuitBreaker
}

func NewAnalyticsClient(serviceURL string) (*AnalyticsClient, error) {
	conn, err := grpc.NewClient(
		serviceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to analytics service: %w", err)
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "AnalyticsService",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	})

	log.Printf("Connected to Analytics Service at %s", serviceURL)

	return &AnalyticsClient{
		conn:   conn,
		client: pb.NewAnalyticsServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *AnalyticsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *AnalyticsClient) GetDailyReport(ctx context.Context, req *pb.DailyReportRequest) (*pb.DailyReportResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetDailyReport(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.DailyReportResponse), nil
}

func (c *AnalyticsClient) GetPeriodReport(ctx context.Context, req *pb.PeriodReportRequest) (*pb.PeriodReportResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetPeriodReport(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.PeriodReportResponse), nil
}

func (c *AnalyticsClient) GetCategoryStats(ctx context.Context, req *pb.CategoryStatsRequest) (*pb.CategoryStatsResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetCategoryStats(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.CategoryStatsResponse), nil
}

func (c *AnalyticsClient) GetTrend(ctx context.Context, req *pb.TrendRequest) (*pb.TrendResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetTrend(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.TrendResponse), nil
}

func (c *AnalyticsClient) GetMonthlyComparison(ctx context.Context, req *pb.MonthlyComparisonRequest) (*pb.MonthlyComparisonResponse, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.GetMonthlyComparison(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pb.MonthlyComparisonResponse), nil
}
