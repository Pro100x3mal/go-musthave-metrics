package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	mocksvc "github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers/mocks"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/audit"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockAuditManager struct{}

func (m *mockAuditManager) Attach(_ audit.Observer)                           {}
func (m *mockAuditManager) NotifyAll(_ context.Context, _ *models.AuditEvent) {}
func (m *mockAuditManager) HasObservers() bool                                { return false }

func setupTestHandler(t *testing.T, setupMock func(*mocksvc.MockMetricsServiceInterface)) http.Handler {
	ctrl := gomock.NewController(t)
	mockService := mocksvc.NewMockMetricsServiceInterface(ctrl)

	if setupMock != nil {
		setupMock(mockService)
	}

	zl := zap.NewNop()
	cfg := &configs.ServerConfig{}
	mockAud := &mockAuditManager{}
	handler := NewMetricsHandler(mockService, zl, cfg, mockAud, nil)

	r := chi.NewRouter()
	initRoutes(r, handler)
	return r
}

func TestUpdateHandler(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		method     string
		wantStatus int
		setupMock  func(*mocksvc.MockMetricsServiceInterface)
	}{
		{
			name:       "non-existent path",
			url:        "/nonexistent/path",
			method:     http.MethodPost,
			wantStatus: http.StatusNotFound,
			setupMock:  func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:       "update counter - method not allowed",
			url:        "/update/counter/test/123",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
			setupMock:  func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:       "update counter - success",
			url:        "/update/counter/test/123",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateMetricFromParams(gomock.Any(), "counter", "test", "123").
					Return(nil)
			},
		},
		{
			name:       "update counter - invalid value",
			url:        "/update/counter/test/abc",
			method:     http.MethodPost,
			wantStatus: http.StatusBadRequest,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateMetricFromParams(gomock.Any(), "counter", "test", "abc").
					Return(models.ErrInvalidMetricValue)
			},
		},
		{
			name:       "update - unsupported type",
			url:        "/update/unknown/test/100",
			method:     http.MethodPost,
			wantStatus: http.StatusBadRequest,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateMetricFromParams(gomock.Any(), "unknown", "test", "100").
					Return(models.ErrUnsupportedMetricType)
			},
		},
		{
			name:       "update gauge - success",
			url:        "/update/gauge/test/3.14",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateMetricFromParams(gomock.Any(), "gauge", "test", "3.14").
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestHandler(t, tt.setupMock)
			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(tt.method, ts.URL+tt.url, nil)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestUpdateJSONHandler(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		setupMock   func(*mocksvc.MockMetricsServiceInterface)
	}{
		{
			name:        "invalid content type",
			contentType: "text/plain",
			body:        `{"id":"test","type":"counter","delta":10}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "invalid JSON",
			contentType: "application/json",
			body:        `{invalid json}`,
			wantStatus:  http.StatusBadRequest,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "missing metric ID",
			contentType: "application/json",
			body:        `{"type":"counter","delta":10}`,
			wantStatus:  http.StatusBadRequest,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "unsupported metric type",
			contentType: "application/json",
			body:        `{"id":"test","type":"unknown","delta":10}`,
			wantStatus:  http.StatusBadRequest,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateJSONMetric(gomock.Any(), gomock.Any()).
					Return(models.ErrUnsupportedMetricType)
			},
		},
		{
			name:        "update counter JSON - success",
			contentType: "application/json",
			body:        `{"id":"test","type":"counter","delta":10}`,
			wantStatus:  http.StatusOK,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateJSONMetric(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:        "update gauge JSON - success",
			contentType: "application/json",
			body:        `{"id":"test","type":"gauge","value":3.14}`,
			wantStatus:  http.StatusOK,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateJSONMetric(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestHandler(t, tt.setupMock)
			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/update/", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", tt.contentType)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestUpdateBatchJSONHandler(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		setupMock   func(*mocksvc.MockMetricsServiceInterface)
	}{
		{
			name:        "invalid content type",
			contentType: "text/plain",
			body:        `[{"id":"counter1","type":"counter","delta":10}]`,
			wantStatus:  http.StatusUnsupportedMediaType,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "invalid JSON",
			contentType: "application/json",
			body:        `[{invalid json}]`,
			wantStatus:  http.StatusBadRequest,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "missing metric ID in batch",
			contentType: "application/json",
			body:        `[{"type":"counter","delta":10}]`,
			wantStatus:  http.StatusBadRequest,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "update batch - metric not found",
			contentType: "application/json",
			body:        `[{"id":"test","type":"counter","delta":5},{"id":"test2","type":"gauge","value":3.14}]`,
			wantStatus:  http.StatusNotFound,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateJSONMetrics(gomock.Any(), gomock.Any()).
					Return(models.ErrMetricNotFound)
			},
		},
		{
			name:        "update batch - success",
			contentType: "application/json",
			body:        `[{"id":"counter1","type":"counter","delta":10},{"id":"gauge1","type":"gauge","value":3.14}]`,
			wantStatus:  http.StatusOK,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					UpdateJSONMetrics(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestHandler(t, tt.setupMock)
			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/updates/", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", tt.contentType)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantBody   string
		setupMock  func(*mocksvc.MockMetricsServiceInterface)
	}{
		{
			name:       "get - not found",
			url:        "/value/counter/unknown",
			wantStatus: http.StatusNotFound,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetMetricValue(gomock.Any(), "counter", "unknown").
					Return("", models.ErrMetricNotFound)
			},
		},
		{
			name:       "get - unsupported type",
			url:        "/value/unknown/test",
			wantStatus: http.StatusBadRequest,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetMetricValue(gomock.Any(), "unknown", "test").
					Return("", models.ErrUnsupportedMetricType)
			},
		},
		{
			name:       "get counter - success",
			url:        "/value/counter/test",
			wantStatus: http.StatusOK,
			wantBody:   "42",
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetMetricValue(gomock.Any(), "counter", "test").
					Return("42", nil)
			},
		},
		{
			name:       "get gauge - success",
			url:        "/value/gauge/test",
			wantStatus: http.StatusOK,
			wantBody:   "3.14",
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetMetricValue(gomock.Any(), "gauge", "test").
					Return("3.14", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestHandler(t, tt.setupMock)
			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := ts.Client().Get(ts.URL + tt.url)
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

func TestGetJSONMetricHandler(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		setupMock   func(*mocksvc.MockMetricsServiceInterface)
		checkBody   func(*testing.T, string)
	}{
		{
			name:        "invalid content type",
			contentType: "text/plain",
			body:        `{"id":"test","type":"counter"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "invalid JSON",
			contentType: "application/json",
			body:        `{invalid json}`,
			wantStatus:  http.StatusBadRequest,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "missing metric ID",
			contentType: "application/json",
			body:        `{"type":"counter"}`,
			wantStatus:  http.StatusBadRequest,
			setupMock:   func(m *mocksvc.MockMetricsServiceInterface) {},
		},
		{
			name:        "get unknown JSON - metric not found",
			contentType: "application/json",
			body:        `{"id":"unknown","type":"counter"}`,
			wantStatus:  http.StatusNotFound,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetJSONMetricValue(gomock.Any(), gomock.Any()).
					Return(nil, models.ErrMetricNotFound)
			},
		},
		{
			name:        "get counter JSON - success",
			contentType: "application/json",
			body:        `{"id":"test","type":"counter"}`,
			wantStatus:  http.StatusOK,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetJSONMetricValue(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, metric *models.Metrics) (*models.Metrics, error) {
						delta := int64(42)
						metric.Delta = &delta
						return metric, nil
					})
			},
			checkBody: func(t *testing.T, body string) {
				var metric models.Metrics
				err := json.Unmarshal([]byte(body), &metric)
				require.NoError(t, err)
				assert.Equal(t, "test", metric.ID)
				assert.Equal(t, "counter", metric.MType)
				assert.NotNil(t, metric.Delta)
				assert.Equal(t, int64(42), *metric.Delta)
			},
		},
		{
			name:        "get gauge JSON - success",
			contentType: "application/json",
			body:        `{"id":"test","type":"gauge"}`,
			wantStatus:  http.StatusOK,
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetJSONMetricValue(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, metric *models.Metrics) (*models.Metrics, error) {
						value := 3.14
						metric.Value = &value
						return metric, nil
					})
			},
			checkBody: func(t *testing.T, body string) {
				var metric models.Metrics
				err := json.Unmarshal([]byte(body), &metric)
				require.NoError(t, err)
				assert.Equal(t, "test", metric.ID)
				assert.Equal(t, "gauge", metric.MType)
				assert.NotNil(t, metric.Value)
				assert.Equal(t, 3.14, *metric.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestHandler(t, tt.setupMock)
			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/value/", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", tt.contentType)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.checkBody != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				tt.checkBody(t, string(body))
			}
		})
	}
}

func TestListAllMetricsHandler(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*mocksvc.MockMetricsServiceInterface)
		wantStatus   int
		wantContains []string
	}{
		{
			name: "list all metrics - error",
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetAllMetrics(gomock.Any()).
					Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "list all metrics - empty",
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetAllMetrics(gomock.Any()).
					Return(map[string]string{}, nil)
			},
			wantStatus:   http.StatusOK,
			wantContains: []string{"<ul>"},
		},
		{
			name: "list all metrics - success",
			setupMock: func(m *mocksvc.MockMetricsServiceInterface) {
				m.EXPECT().
					GetAllMetrics(gomock.Any()).
					Return(map[string]string{
						"test_counter": "42",
						"test_gauge":   "3.14",
					}, nil)
			},
			wantStatus: http.StatusOK,
			wantContains: []string{
				"<li>test_counter: 42</li>",
				"<li>test_gauge: 3.14</li>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestHandler(t, tt.setupMock)
			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := ts.Client().Get(ts.URL + "/")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			html := string(body)

			for _, want := range tt.wantContains {
				assert.Contains(t, html, want)
			}
		})
	}
}
