package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/service"
	"github.com/stretchr/testify/assert"
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

func TestUpdateMetricsHandler(t *testing.T) {
	h := newMetricsHandler(&mockUpdater{})
	tests := []struct {
		url        string
		method     string
		wantStatus int
	}{
		{"/update/counter/test/123", http.MethodPost, http.StatusOK},
		{"/update/counter/test/-321", http.MethodPost, http.StatusOK},
		{"/update/counter/test/123.123", http.MethodPost, http.StatusBadRequest},
		{"/update/counter/test/123a", http.MethodPost, http.StatusBadRequest},
		{"/update/unknown/test/100", http.MethodPost, http.StatusBadRequest},
		{"/update/gauge/test/123.321", http.MethodPost, http.StatusOK},
		{"/update/gauge/test/-321.123", http.MethodPost, http.StatusOK},
		{"/update/gauge/test/123", http.MethodPost, http.StatusOK},
		{"/update/gauge/test/123a", http.MethodPost, http.StatusBadRequest},
		{"/update/unknown/test/100.01", http.MethodPost, http.StatusBadRequest},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.url, nil)
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		h.UpdateMetricsHandler(w, req)

		assert.Equal(t, w.Result().StatusCode, tt.wantStatus, "URL %q method %s: expected status %d, got %d", tt.url, tt.method, tt.wantStatus, w.Result().StatusCode)

	}
}

func TestUpdateMetricsHandler_WrongContentType(t *testing.T) {
	h := newMetricsHandler(&mockUpdater{})

	req := httptest.NewRequest(http.MethodPost, "/update/counter/test/100", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateMetricsHandler(w, req)

	assert.Equal(t, w.Result().StatusCode, http.StatusUnsupportedMediaType, "expected status %d, got %d", http.StatusUnsupportedMediaType, w.Result().StatusCode)
}
