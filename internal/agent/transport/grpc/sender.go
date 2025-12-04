package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/transport"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Sender struct {
	conn     *grpc.ClientConn
	client   proto.MetricsClient
	provider transport.MetricsProvider
	logger   *zap.Logger
	ip       string
}

func NewSender(addr string, provider transport.MetricsProvider, logger *zap.Logger) (*Sender, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	ip, err := getOutboundIP()
	if err != nil {
		logger.Warn("Failed to get local IP, using empty IP", zap.Error(err))
		ip = ""
	}

	return &Sender{
		conn:     conn,
		client:   proto.NewMetricsClient(conn),
		provider: provider,
		logger:   logger,
		ip:       ip,
	}, nil
}

func (s *Sender) SendMetrics(ctx context.Context) error {
	metrics := s.provider.GetAllMetrics()
	if len(metrics) == 0 {
		s.logger.Debug("No metrics to send")
		return nil
	}

	protoMetrics := make([]*proto.Metric, 0, len(metrics))
	for _, m := range metrics {
		protoMetric := &proto.Metric{
			Id: m.ID,
		}

		switch m.MType {
		case models.Gauge:
			protoMetric.Type = proto.Metric_GAUGE
			if m.Value != nil {
				protoMetric.Value = *m.Value
			}
		case models.Counter:
			protoMetric.Type = proto.Metric_COUNTER
			if m.Delta != nil {
				protoMetric.Delta = *m.Delta
			}
		default:
			s.logger.Warn("Unknown metric type", zap.String("id", m.ID), zap.String("type", m.MType))
			continue
		}

		protoMetrics = append(protoMetrics, protoMetric)
	}

	s.logger.Debug("Sending metrics via gRPC", zap.Int("count", len(protoMetrics)))

	md := metadata.New(map[string]string{
		"x-real-ip": s.ip,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req := &proto.UpdateMetricsRequest{
		Metrics: protoMetrics,
	}

	_, err := s.client.UpdateMetrics(reqCtx, req)
	if err != nil {
		return fmt.Errorf("failed to send metrics via gRPC: %w", err)
	}

	s.logger.Info("Successfully sent metrics via gRPC", zap.Int("count", len(protoMetrics)))
	return nil
}

func (s *Sender) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func getOutboundIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no non-loopback IP address found")
}
