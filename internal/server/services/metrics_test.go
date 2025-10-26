package services

import (
	"context"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	mocksrepo "github.com/Pro100x3mal/go-musthave-metrics/internal/server/services/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestMetricsService_UpdateMetricFromParams(t *testing.T) {
	tests := []struct {
		name      string
		mType     string
		mName     string
		mValue    string
		setupMock func(*mocksrepo.MockRepository)
		wantErr   error
	}{
		{
			name:      "unknown metric type",
			mType:     "unknown",
			mName:     "test",
			mValue:    "100",
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantErr:   models.ErrUnsupportedMetricType,
		},
		{
			name:      "counter - invalid float value",
			mType:     "counter",
			mName:     "test",
			mValue:    "123.45",
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantErr:   models.ErrInvalidMetricValue,
		},
		{
			name:      "counter - invalid non-numeric value",
			mType:     "counter",
			mName:     "test",
			mValue:    "abc",
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantErr:   models.ErrInvalidMetricValue,
		},
		{
			name:   "counter - valid positive value",
			mType:  "counter",
			mName:  "test",
			mValue: "123",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateCounter(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:   "counter - valid negative value",
			mType:  "counter",
			mName:  "test",
			mValue: "-321",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateCounter(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:      "gauge - invalid non-numeric value",
			mType:     "gauge",
			mName:     "test",
			mValue:    "abc",
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantErr:   models.ErrInvalidMetricValue,
		},
		{
			name:   "gauge - valid float value",
			mType:  "gauge",
			mName:  "test",
			mValue: "3.14",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateGauge(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:   "gauge - valid integer value",
			mType:  "gauge",
			mName:  "test",
			mValue: "123",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateGauge(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:   "gauge - valid negative value",
			mType:  "gauge",
			mName:  "test",
			mValue: "-123.456",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateGauge(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocksrepo.NewMockRepository(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewMetricsService(mockRepo)

			err := service.UpdateMetricFromParams(context.Background(), tt.mType, tt.mName, tt.mValue)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricsService_UpdateJSONMetric(t *testing.T) {
	tests := []struct {
		name      string
		metric    *models.Metrics
		setupMock func(*mocksrepo.MockRepository)
		wantErr   error
	}{
		{
			name:      "nil metric",
			metric:    nil,
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantErr:   models.ErrMetricNotFound,
		},
		{
			name: "unknown type",
			metric: &models.Metrics{
				ID:    "test",
				MType: "unknown",
			},
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantErr:   models.ErrUnsupportedMetricType,
		},
		{
			name: "counter - success",
			metric: &models.Metrics{
				ID:    "test",
				MType: "counter",
				Delta: int64Ptr(10),
			},
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateCounter(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "gauge - success",
			metric: &models.Metrics{
				ID:    "test",
				MType: "gauge",
				Value: float64Ptr(3.14),
			},
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateGauge(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocksrepo.NewMockRepository(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewMetricsService(mockRepo)

			err := service.UpdateJSONMetric(context.Background(), tt.metric)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricsService_UpdateJSONMetrics(t *testing.T) {
	tests := []struct {
		name      string
		metrics   []models.Metrics
		setupMock func(*mocksrepo.MockRepository)
		wantErr   error
	}{
		{
			name:      "empty metrics",
			metrics:   nil,
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantErr:   models.ErrMetricNotFound,
		},
		{
			name: "valid metrics",
			metrics: []models.Metrics{
				{
					ID:    "test",
					MType: "gauge",
					Value: float64Ptr(3.14),
				},
				{
					ID:    "test2",
					MType: "counter",
					Delta: int64Ptr(10),
				},
			},
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					UpdateMetrics(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocksrepo.NewMockRepository(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewMetricsService(mockRepo)

			err := service.UpdateJSONMetrics(context.Background(), tt.metrics)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricsService_GetMetricValue(t *testing.T) {
	tests := []struct {
		name      string
		mType     string
		mName     string
		setupMock func(*mocksrepo.MockRepository)
		wantValue string
		wantErr   error
	}{
		{
			name:      "unknown type",
			mType:     "unknown",
			mName:     "test",
			setupMock: func(m *mocksrepo.MockRepository) {},
			wantValue: "",
			wantErr:   models.ErrUnsupportedMetricType,
		},
		{
			name:  "counter - not found",
			mType: "counter",
			mName: "unknown",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetCounter(gomock.Any(), "unknown").
					Return(int64(0), models.ErrMetricNotFound)
			},
			wantValue: "",
			wantErr:   models.ErrMetricNotFound,
		},
		{
			name:  "gauge - not found",
			mType: "gauge",
			mName: "unknown",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetGauge(gomock.Any(), "unknown").
					Return(float64(0), models.ErrMetricNotFound)
			},
			wantValue: "",
			wantErr:   models.ErrMetricNotFound,
		},
		{
			name:  "counter - success",
			mType: "counter",
			mName: "test",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetCounter(gomock.Any(), "test").
					Return(int64(42), nil)
			},
			wantValue: "42",
			wantErr:   nil,
		},
		{
			name:  "gauge - success",
			mType: "gauge",
			mName: "test",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetGauge(gomock.Any(), "test").
					Return(3.14, nil)
			},
			wantValue: "3.14",
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocksrepo.NewMockRepository(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewMetricsService(mockRepo)

			value, err := service.GetMetricValue(context.Background(), tt.mType, tt.mName)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}
}

func TestMetricsService_GetJSONMetricValue(t *testing.T) {
	tests := []struct {
		name       string
		metric     *models.Metrics
		setupMock  func(*mocksrepo.MockRepository)
		wantMetric *models.Metrics
		wantErr    error
	}{
		{
			name:       "nil metric",
			metric:     nil,
			setupMock:  func(m *mocksrepo.MockRepository) {},
			wantMetric: nil,
			wantErr:    models.ErrMetricNotFound,
		},
		{
			name: "unknown type",
			metric: &models.Metrics{
				ID:    "test",
				MType: "unknown",
			},
			setupMock:  func(m *mocksrepo.MockRepository) {},
			wantMetric: nil,
			wantErr:    models.ErrUnsupportedMetricType,
		},
		{
			name: "counter - not found",
			metric: &models.Metrics{
				ID:    "test",
				MType: "counter",
			},
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetCounter(gomock.Any(), "test").
					Return(int64(0), models.ErrMetricNotFound)
			},
			wantMetric: nil,
			wantErr:    models.ErrMetricNotFound,
		},
		{
			name: "gauge - not found",
			metric: &models.Metrics{
				ID:    "test",
				MType: "gauge",
			},
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetGauge(gomock.Any(), "test").
					Return(float64(0), models.ErrMetricNotFound)
			},
			wantMetric: nil,
			wantErr:    models.ErrMetricNotFound,
		},
		{
			name: "counter - success",
			metric: &models.Metrics{
				ID:    "test",
				MType: "counter",
				Delta: int64Ptr(42),
			},
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetCounter(gomock.Any(), "test").
					Return(int64(42), nil)
			},
			wantMetric: &models.Metrics{
				ID:    "test",
				MType: "counter",
				Delta: int64Ptr(42),
			},
			wantErr: nil,
		},
		{
			name: "gauge - success",
			metric: &models.Metrics{
				ID:    "test",
				MType: "gauge",
				Value: float64Ptr(3.14),
			},
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetGauge(gomock.Any(), "test").
					Return(3.14, nil)
			},
			wantMetric: &models.Metrics{
				ID:    "test",
				MType: "gauge",
				Value: float64Ptr(3.14),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocksrepo.NewMockRepository(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewMetricsService(mockRepo)

			metric, err := service.GetJSONMetricValue(context.Background(), tt.metric)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMetric, metric)
			}
		})
	}
}

func TestMetricsService_GetAllMetrics(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocksrepo.MockRepository)
		wantMap   map[string]string
		wantErr   error
	}{
		{
			name: "empty - no gauges",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetAllGauges(gomock.Any()).
					Return(nil, models.ErrMetricNotFound)
			},
			wantMap: nil,
			wantErr: models.ErrMetricNotFound,
		},
		{
			name: "empty - no counters",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetAllGauges(gomock.Any()).
					Return(map[string]float64{}, nil)
				m.EXPECT().
					GetAllCounters(gomock.Any()).
					Return(nil, models.ErrMetricNotFound)
			},
			wantMap: nil,
			wantErr: models.ErrMetricNotFound,
		},
		{
			name: "success - both gauges and counters",
			setupMock: func(m *mocksrepo.MockRepository) {
				m.EXPECT().
					GetAllGauges(gomock.Any()).
					Return(map[string]float64{
						"gauge1": 3.14,
						"gauge2": 2.71,
					}, nil)
				m.EXPECT().
					GetAllCounters(gomock.Any()).
					Return(map[string]int64{
						"counter1": 42,
						"counter2": 100,
					}, nil)
			},
			wantMap: map[string]string{
				"gauge1":   "3.14",
				"gauge2":   "2.71",
				"counter1": "42",
				"counter2": "100",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocksrepo.NewMockRepository(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewMetricsService(mockRepo)

			result, err := service.GetAllMetrics(context.Background())

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMap, result)
			}
		})
	}
}
