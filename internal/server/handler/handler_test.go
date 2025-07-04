package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repository"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUpdater struct{}

func (m *mockUpdater) UpdateMetricFromParams(mType, mName, mValue string) error {
	switch mType {
	case "counter":
		switch mValue {
		case "123", "-321":
			return nil
		case "123.123":
			return service.ErrInvalidMetricValue
		case "123a":
			return service.ErrInvalidMetricValue
		default:
			return service.ErrInvalidMetricValue
		}
	case "gauge":
		switch mValue {
		case "123.321", "-321.123", "123":
			return nil
		case "123.123":
			return service.ErrInvalidMetricValue
		case "123a":
			return service.ErrInvalidMetricValue
		default:
			return service.ErrInvalidMetricValue
		}
	default:
		return service.ErrUnsupportedMetricType
	}
}

func (m *mockUpdater) GetMetricValue(mType, mName string) (string, error) {
	if mName != "test" {
		return "", repository.ErrMetricNotFound
	}
	switch mType {
	case "counter":
		return "42", nil
	case "gauge":
		return "3.14", nil
	default:
		return "", service.ErrUnsupportedMetricType
	}
}

func TestUpdateMetricsHandler(t *testing.T) {
	h := newMetricsHandler(&mockUpdater{})
	r := newRouter(h)
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

func TestUpdateMetricsHandler_WrongContentType(t *testing.T) {
	h := newMetricsHandler(&mockUpdater{})
	r := newRouter(h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/update/counter/test/100", nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode, "expected status %d, got %d", http.StatusUnsupportedMediaType, resp.StatusCode)
}

func TestGetMetricValueHandler(t *testing.T) {
	h := newMetricsHandler(&mockUpdater{})
	r := newRouter(h)
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
