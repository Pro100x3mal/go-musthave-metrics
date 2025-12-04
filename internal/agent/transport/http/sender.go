package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/transport"
	"github.com/Pro100x3mal/go-musthave-metrics/pkg/crypto"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Sender struct {
	client   *resty.Client
	provider transport.MetricsProvider
	logger   *zap.Logger
}

func NewSender(cfg *configs.AgentConfig, publicKey *rsa.PublicKey, provider transport.MetricsProvider, logger *zap.Logger) *Sender {
	c := resty.New().
		SetBaseURL("http://" + cfg.ServerAddr).
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	c.OnBeforeRequest(func(_ *resty.Client, r *resty.Request) error {
		realIP, err := getOutboundIP()
		if err != nil {
			logger.Warn("Failed to get local IP", zap.Error(err))
		} else {
			r.SetHeader("X-Real-IP", realIP)
		}

		if body, ok := r.Body.([]byte); ok && len(body) > 0 {
			newBody, hash, err := prepareRequestData(body, publicKey, cfg.Key)
			if err != nil {
				return err
			}

			if publicKey != nil {
				r.SetBody(newBody)
			}

			if cfg.Key != "" {
				r.SetHeader("HashSHA256", hash)
			}
		}
		return nil
	})

	return &Sender{
		client:   c,
		provider: provider,
		logger:   logger,
	}
}

func (s *Sender) SendMetrics(ctx context.Context) error {
	metrics := s.provider.GetAllMetrics()
	if len(metrics) == 0 {
		return errors.New("no metrics to send")
	}

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	err := json.NewEncoder(gz).Encode(metrics)
	if err != nil {
		return fmt.Errorf("gzip encoding failed: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	_, err = s.client.R().
		SetContext(ctx).
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Content-Type", "application/json").
		SetBody(buf.Bytes()).
		Post("/updates/")

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	s.logger.Info("Successfully sent metrics via HTTP", zap.Int("count", len(metrics)))
	return nil
}

func prepareRequestData(body []byte, publicKey *rsa.PublicKey, key string) ([]byte, string, error) {
	if publicKey != nil {
		var err error
		body, err = crypto.Encrypt(publicKey, body)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encrypt request body: %w", err)
		}
	}

	var hash string
	if key != "" {
		hash = signBody(body, key)
	}

	return body, hash, nil
}

func signBody(body []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
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
