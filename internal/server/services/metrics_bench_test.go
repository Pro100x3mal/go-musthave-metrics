package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
)

func generateBenchMetrics(n int) []models.Metrics {
	metrics := make([]models.Metrics, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			value := float64(i) * 1.5
			metrics[i] = models.Metrics{
				ID:    fmt.Sprintf("gauge_%d", i),
				MType: models.Gauge,
				Value: &value,
			}
		} else {
			delta := int64(i)
			metrics[i] = models.Metrics{
				ID:    fmt.Sprintf("counter_%d", i),
				MType: models.Counter,
				Delta: &delta,
			}
		}
	}
	return metrics
}

func BenchmarkUpdateMetricFromParams(b *testing.B) {
	repo := repositories.NewMemStorage()
	service := NewMetricsService(repo)
	ctx := context.Background()

	b.Run("gauge", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = service.UpdateMetricFromParams(ctx, models.Gauge, "test", "42.5")
		}
	})

	b.Run("counter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = service.UpdateMetricFromParams(ctx, models.Counter, "test", "100")
		}
	})
}

func BenchmarkUpdateJSONMetric(b *testing.B) {
	repo := repositories.NewMemStorage()
	service := NewMetricsService(repo)
	ctx := context.Background()

	value := 42.5
	metric := &models.Metrics{
		ID:    "test_gauge",
		MType: models.Gauge,
		Value: &value,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = service.UpdateJSONMetric(ctx, metric)
	}
}

func BenchmarkUpdateJSONMetrics(b *testing.B) {
	repo := repositories.NewMemStorage()
	service := NewMetricsService(repo)
	ctx := context.Background()

	sizes := []int{10, 100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			metrics := generateBenchMetrics(size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = service.UpdateJSONMetrics(ctx, metrics)
			}
		})
	}
}

func BenchmarkGetMetricValue_Gauge(b *testing.B) {
	repo := repositories.NewMemStorage()
	service := NewMetricsService(repo)
	ctx := context.Background()

	value := 42.5
	metric := &models.Metrics{
		ID:    "test_gauge",
		MType: models.Gauge,
		Value: &value,
	}
	_ = service.UpdateJSONMetric(ctx, metric)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = service.GetMetricValue(ctx, models.Gauge, "test_gauge")
	}
}

func BenchmarkGetMetricValue_Counter(b *testing.B) {
	repo := repositories.NewMemStorage()
	service := NewMetricsService(repo)
	ctx := context.Background()

	delta := int64(100)
	metric := &models.Metrics{
		ID:    "test_counter",
		MType: models.Counter,
		Delta: &delta,
	}
	_ = service.UpdateJSONMetric(ctx, metric)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = service.GetMetricValue(ctx, models.Counter, "test_counter")
	}
}

func BenchmarkGetJSONMetricValue(b *testing.B) {
	repo := repositories.NewMemStorage()
	service := NewMetricsService(repo)
	ctx := context.Background()

	value := 42.5
	metric := &models.Metrics{
		ID:    "test_gauge",
		MType: models.Gauge,
		Value: &value,
	}
	_ = service.UpdateJSONMetric(ctx, metric)

	requestMetric := &models.Metrics{
		ID:    "test_gauge",
		MType: models.Gauge,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = service.GetJSONMetricValue(ctx, requestMetric)
	}
}

func BenchmarkGetAllMetrics(b *testing.B) {
	ctx := context.Background()

	sizes := []int{10, 100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			repo := repositories.NewMemStorage()
			service := NewMetricsService(repo)

			metrics := generateBenchMetrics(size)
			for _, m := range metrics {
				if m.MType == models.Gauge {
					_ = repo.UpdateGauge(ctx, &m)
				} else {
					_ = repo.UpdateCounter(ctx, &m)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = service.GetAllMetrics(ctx)
			}
		})
	}
}
