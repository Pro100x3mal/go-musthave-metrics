package services

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
	"sync"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
	"github.com/Pro100x3mal/go-musthave-metrics/pkg/crypto"
	"github.com/go-resty/resty/v2"
)

type RepositoryReader interface {
	GetAllMetrics() []*models.Metrics
}

type MetricsQueryService struct {
	reader RepositoryReader
}

func NewMetricsQueryService(reader RepositoryReader) *MetricsQueryService {
	return &MetricsQueryService{
		reader: reader,
	}
}

type Client struct {
	client *resty.Client
}

func NewClient(cfg *configs.AgentConfig, publicKey *rsa.PublicKey) *Client {
	c := resty.New().
		SetBaseURL("http://" + cfg.ServerAddr).
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	c.OnBeforeRequest(func(_ *resty.Client, r *resty.Request) error {
		if body, ok := r.Body.([]byte); ok && len(body) > 0 {
			var err error

			if publicKey != nil {
				body, err = crypto.Encrypt(publicKey, body)
				if err != nil {
					return fmt.Errorf("failed to encrypt request body: %w", err)
				}
				r.SetBody(body)
			}

			if cfg.Key != "" {
				r.SetHeader("HashSHA256", signBody(body, cfg.Key))
			}
		}
		return nil
	})

	return &Client{
		client: c,
	}
}

func (qs *MetricsQueryService) SendMetrics(ctx context.Context, c *Client) error {
	metrics := qs.reader.GetAllMetrics()
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

	_, err = c.client.R().
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

	return nil
}

func signBody(body []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

type Task func()

type WorkerPool struct {
	numWorkers int
	queue      chan Task
	wg         sync.WaitGroup
}

func NewWorkerPool(cfg *configs.AgentConfig) *WorkerPool {
	numWorkers := cfg.RateLimit
	if numWorkers <= 0 {
		numWorkers = 1
	}

	return &WorkerPool{
		numWorkers: numWorkers,
		queue:      make(chan Task, numWorkers),
	}
}

func (p *WorkerPool) Start() {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for task := range p.queue {
				task()
			}
		}()
	}
}

func (p *WorkerPool) Submit(t Task) {
	p.queue <- t
}

func (p *WorkerPool) Stop() {
	close(p.queue)
	p.wg.Wait()
}
