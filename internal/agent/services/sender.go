package services

import (
	"sync"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
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

func (qs *MetricsQueryService) GetAllMetrics() []*models.Metrics {
	return qs.reader.GetAllMetrics()
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
