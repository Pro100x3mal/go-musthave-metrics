package repositories

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
)

func generateBenchMaps(n int) map[string]float64 {
	gaugeMap := make(map[string]float64)
	for i := 0; i < n; i++ {
		gaugeMap[fmt.Sprintf("gauge_%d", i)] = float64(i) * 2
	}
	return gaugeMap
}

func generateBenchMetrics(n int) []models.Metrics {
	gauges := make([]models.Metrics, n)
	for i := 0; i < n; i++ {
		value := float64(i) * 2
		gauges[i] = models.Metrics{
			ID:    fmt.Sprintf("gauge_%d", i),
			Value: &value,
			MType: models.Gauge,
		}
	}
	return gauges
}

func BenchmarkUpdateMetrics_WithoutCapacity(b *testing.B) {
	sizes := []int{100, 1000, 10000, 1000000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			gaugeMap := generateBenchMaps(size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var gauges []models.Metrics
				for id, v := range gaugeMap {
					value := v
					gauges = append(gauges, models.Metrics{
						ID:    id,
						Value: &value,
						MType: models.Gauge,
					})
				}
				_ = gauges
			}
		})
	}
}

func BenchmarkUpdateMetrics_WithCapacity(b *testing.B) {
	sizes := []int{100, 1000, 10000, 1000000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			gaugeMap := generateBenchMaps(size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				gauges := make([]models.Metrics, 0, len(gaugeMap))
				for id, v := range gaugeMap {
					value := v
					gauges = append(gauges, models.Metrics{
						ID:    id,
						Value: &value,
						MType: models.Gauge,
					})
				}
				_ = gauges
			}
		})
	}
}

func BenchmarkSQLBuilding_StringConcat(b *testing.B) {
	sizes := []int{100, 1000, 10000, 1000000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			gauges := generateBenchMetrics(size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				values := make([]string, 0, len(gauges))
				args := make([]any, 0, len(gauges)*2)

				for j, m := range gauges {
					base := j * 2
					params := fmt.Sprintf("($%d, $%d)", base+1, base+2)
					values = append(values, params)
					args = append(args, m.ID, *m.Value)
				}

				_ = `INSERT INTO gauges (id, value) VALUES ` + strings.Join(values, ",") + ` ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value`
				_ = args
			}
		})
	}
}

func BenchmarkSQLBuilding_StringBuilder(b *testing.B) {
	sizes := []int{100, 1000, 10000, 1000000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			gauges := generateBenchMetrics(size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				values := make([]string, 0, len(gauges))
				args := make([]any, 0, len(gauges)*2)

				var queryBuilder strings.Builder
				queryBuilder.WriteString("INSERT INTO gauges (id, value) VALUES ")

				for j, m := range gauges {
					base := j * 2
					params := fmt.Sprintf("($%d, $%d)", base+1, base+2)
					values = append(values, params)
					args = append(args, m.ID, *m.Value)
				}

				queryBuilder.WriteString(strings.Join(values, ","))
				queryBuilder.WriteString(" ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value")

				_ = queryBuilder.String()
				_ = args
			}
		})
	}
}
