package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/proto"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/ipfilter"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type MetricsServiceWriter interface {
	UpdateJSONMetrics(ctx context.Context, metrics []models.Metrics) error
}

type MetricsServer struct {
	proto.UnimplementedMetricsServer
	metricService MetricsServiceWriter
	logger        *zap.Logger
}

func NewMetricsServer(metricService MetricsServiceWriter, logger *zap.Logger) *MetricsServer {
	return &MetricsServer{
		metricService: metricService,
		logger:        logger,
	}
}

func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	s.logger.Info("Received UpdateMetrics request", zap.Int("metrics_count", len(req.Metrics)))

	var metrics []models.Metrics
	for _, protoMetric := range req.Metrics {
		metric := models.Metrics{
			ID: protoMetric.Id,
		}

		switch protoMetric.Type {
		case proto.Metric_GAUGE:
			metric.MType = models.Gauge
			value := protoMetric.Value
			metric.Value = &value
		case proto.Metric_COUNTER:
			metric.MType = models.Counter
			delta := protoMetric.Delta
			metric.Delta = &delta
		default:
			s.logger.Warn("Unknown metric type", zap.String("id", protoMetric.Id), zap.Int32("type", int32(protoMetric.Type)))
			continue
		}

		metrics = append(metrics, metric)
	}

	if err := s.metricService.UpdateJSONMetrics(ctx, metrics); err != nil {
		s.logger.Error("Failed to update metrics", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update metrics: %v", err)
	}

	s.logger.Info("Successfully updated metrics", zap.Int("count", len(metrics)))
	return &proto.UpdateMetricsResponse{}, nil
}

func IPFilterInterceptor(filter *ipfilter.IPFilter, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !filter.IsEnabled() {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("No metadata in request")
			return nil, status.Error(codes.PermissionDenied, "no metadata provided")
		}

		ips := md.Get("x-real-ip")
		if len(ips) == 0 {
			logger.Warn("No x-real-ip in metadata")
			return nil, status.Error(codes.PermissionDenied, "no IP address provided")
		}

		if err := filter.ValidateIP(ips[0]); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		return handler(ctx, req)
	}
}

func StartGRPCServer(ctx context.Context, addr string, filter *ipfilter.IPFilter, metricsServer *MetricsServer, logger *zap.Logger) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	var opts []grpc.ServerOption
	if filter.IsEnabled() {
		opts = append(opts, grpc.UnaryInterceptor(IPFilterInterceptor(filter, logger)))
	}

	grpcServer := grpc.NewServer(opts...)
	proto.RegisterMetricsServer(grpcServer, metricsServer)

	logger.Info("Starting gRPC server", zap.String("address", addr))

	serverErrCh := make(chan error, 1)

	go func() {
		if err = grpcServer.Serve(listener); err != nil {
			serverErrCh <- fmt.Errorf("failed to serve gRPC: %w", err)
			return
		}
		serverErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("gRPC server is shutting down...")

		shutdownDone := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(shutdownDone)
		}()

		shutdownTimeout := time.After(5 * time.Second)
		select {
		case <-shutdownDone:
			logger.Info("gRPC server shutdown complete")
		case <-shutdownTimeout:
			logger.Warn("gRPC server shutdown timeout exceeded, forcing stop")
			grpcServer.Stop()
		}

		return nil
	case err = <-serverErrCh:
		return err
	}
}
