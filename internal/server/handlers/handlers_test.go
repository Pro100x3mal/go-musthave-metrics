package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/audit"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockUpdater struct{}

type mockAuditManager struct{}

func initRouterForTests() http.Handler {
	mockUpd := &mockUpdater{}
	mockAud := &mockAuditManager{}
	zl := zap.NewNop()
	cfg := &configs.ServerConfig{}
	handler := NewMetricsHandler(mockUpd, zl, cfg, mockAud)

	r := chi.NewRouter()
	initRoutes(r, handler)
	return r
}

func (m *mockAuditManager) Attach(_ audit.Observer) {}

func (m *mockAuditManager) NotifyAll(_ context.Context, _ *models.AuditEvent) {}

func (m *mockAuditManager) HasObservers() bool {
	return false
}

func (m *mockUpdater) UpdateJSONMetrics(_ context.Context, _ []models.Metrics) error {
	return nil
}

func (m *mockUpdater) PingCheck(_ context.Context) error {
	return nil
}

func (m *mockUpdater) GetJSONMetricValue(_ context.Context, metric *models.Metrics) (*models.Metrics, error) {
	return metric, nil
}

func (m *mockUpdater) UpdateJSONMetric(_ context.Context, _ *models.Metrics) error {
	return nil
}

func (m *mockUpdater) UpdateMetricFromParams(_ context.Context, mType, _, mValue string) error {
	switch mType {
	case "counter":
		switch mValue {
		case "123", "-321":
			return nil
		case "123.123":
			return models.ErrInvalidMetricValue
		case "123a":
			return models.ErrInvalidMetricValue
		default:
			return models.ErrInvalidMetricValue
		}
	case "gauge":
		switch mValue {
		case "123.321", "-321.123", "123":
			return nil
		case "123.123":
			return models.ErrInvalidMetricValue
		case "123a":
			return models.ErrInvalidMetricValue
		default:
			return models.ErrInvalidMetricValue
		}
	default:
		return models.ErrUnsupportedMetricType
	}
}

func (m *mockUpdater) GetMetricValue(_ context.Context, mType, mName string) (string, error) {
	if mName != "test" {
		return "", models.ErrMetricNotFound
	}
	switch mType {
	case "counter":
		return "42", nil
	case "gauge":
		return "3.14", nil
	default:
		return "", models.ErrUnsupportedMetricType
	}
}

func (m *mockUpdater) GetAllMetrics(_ context.Context) (map[string]string, error) {
	return map[string]string{
		"test_counter": "42",
		"test_gauge":   "3.14",
	}, nil
}

func TestUpdateMetricsHandler(t *testing.T) {
	r := initRouterForTests()
	ts := httptest.NewServer(r)
	defer ts.Close()

	tests := []struct {
		url        string
		method     string
		wantStatus int
	}{
		{"/nonexistent/path", http.MethodPost, http.StatusNotFound},
		{"/update/counter/test/123", http.MethodPost, http.StatusOK},
		{"/update/counter/test/123", http.MethodGet, http.StatusMethodNotAllowed},
		{"/update/counter/test/-321", http.MethodPost, http.StatusOK},
		{"/update/counter/test/123.123", http.MethodPost, http.StatusBadRequest},
		{"/update/counter/test/123a", http.MethodPost, http.StatusBadRequest},
		{"/update/unknown/test/100", http.MethodPost, http.StatusBadRequest},
		{"/update/gauge/test/123.321", http.MethodPost, http.StatusOK},
		{"/update/gauge/test/123.321", http.MethodGet, http.StatusMethodNotAllowed},
		{"/update/gauge/test/-321.123", http.MethodPost, http.StatusOK},
		{"/update/gauge/test/123", http.MethodPost, http.StatusOK},
		{"/update/gauge/test/123a", http.MethodPost, http.StatusBadRequest},
		{"/update/unknown/test/100.01", http.MethodPost, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, ts.URL+tt.url, nil)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "text/plain")

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode, "URL %q method %s: expected status %d, got %d", tt.url, tt.method, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetMetricValueHandler(t *testing.T) {
	r := initRouterForTests()
	ts := httptest.NewServer(r)
	defer ts.Close()

	tests := []struct {
		url        string
		method     string
		wantStatus int
		wantBody   string
	}{
		{url: "/value/counter/test", method: http.MethodGet, wantStatus: http.StatusOK, wantBody: "42"},
		{url: "/value/gauge/test", method: http.MethodGet, wantStatus: http.StatusOK, wantBody: "3.14"},
		{url: "/value/counter/unknown", method: http.MethodGet, wantStatus: http.StatusNotFound},
		{url: "/value/unknown/test", method: http.MethodGet, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, ts.URL+tt.url, nil)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantStatus == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.wantBody, string(body))
			}
		})
	}
}

func TestListAllMetricsHandler(t *testing.T) {
	r := initRouterForTests()
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	html := string(body)

	assert.Contains(t, html, "<li>test_counter: 42</li>")
	assert.Contains(t, html, "<li>test_gauge: 3.14</li>")
}
