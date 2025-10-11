package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"go.uber.org/zap"
)

type Observer interface {
	Notify(ctx context.Context, event *models.AuditEvent) error
}

type Publisher interface {
	Attach(observer Observer)
	NotifyAll(ctx context.Context, event *models.AuditEvent)
}

type AuditManager struct {
	observers []Observer
	mu        *sync.RWMutex
	logger    *zap.Logger
}

func NewAuditManager(logger *zap.Logger) *AuditManager {
	return &AuditManager{
		observers: make([]Observer, 0),
		mu:        &sync.RWMutex{},
		logger:    logger,
	}
}

func (am *AuditManager) Attach(observer Observer) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.observers = append(am.observers, observer)
}

func (am *AuditManager) NotifyAll(ctx context.Context, event *models.AuditEvent) {
	notifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	am.mu.RLock()
	observers := make([]Observer, len(am.observers))
	copy(observers, am.observers)
	am.mu.RUnlock()

	var wg sync.WaitGroup
	for _, observer := range observers {
		wg.Add(1)
		go func(obs Observer) {
			defer wg.Done()
			if err := obs.Notify(notifyCtx, event); err != nil {
				am.logger.Error("failed to notify audit observer", zap.Error(err))
			}
		}(observer)
	}
	wg.Wait()
}

func (am *AuditManager) HasObservers() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.observers) > 0
}

type FileAuditObserver struct {
	filePath string
	mu       *sync.Mutex
}

func NewFileAuditObserver(filePath string) *FileAuditObserver {
	return &FileAuditObserver{
		filePath: filePath,
		mu:       &sync.Mutex{},
	}
}

func (fao *FileAuditObserver) Notify(_ context.Context, event *models.AuditEvent) error {
	fao.mu.Lock()
	defer fao.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	f, err := os.OpenFile(fao.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}
	defer f.Close()

	if _, err = f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit event: %w", err)
	}

	return nil
}

type HTTPAuditObserver struct {
	url    string
	client *http.Client
}

func NewHTTPAuditObserver(url string) *HTTPAuditObserver {
	return &HTTPAuditObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (hao *HTTPAuditObserver) Notify(ctx context.Context, event *models.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hao.url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create audit request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := hao.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send audit event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit server returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

func NewAuditEventFromMetrics(metrics []models.Metrics, ipAddress string) *models.AuditEvent {
	metricNames := make([]string, 0, len(metrics))
	for _, m := range metrics {
		metricNames = append(metricNames, m.ID)
	}

	return &models.AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: ipAddress,
	}
}

func NewAuditEventFromMetric(metric *models.Metrics, ipAddress string) *models.AuditEvent {
	return &models.AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{metric.ID},
		IPAddress: ipAddress,
	}
}

func GetIPAddress(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
