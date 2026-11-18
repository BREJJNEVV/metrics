// Package handler предоставляет HTTP-обработчики и мидлвары.
package handler

import (
	"context"

	"github.com/BREJJNEVV/metrics/internal/audit"
	"github.com/BREJJNEVV/metrics/internal/model"
	"go.uber.org/zap"
)

// Repository определяет контракт хранилища метрик.
type Repository interface {
	// Set устанавливает абсолютное значение метрики типа Gauge.
	Set(ctx context.Context, name string, value float64) error

	// Add увеличивает значение метрики типа Counter на указанную величину.
	Add(ctx context.Context, name string, value int64) error

	// GetGauge возвращает текущее значение метрики типа Gauge и найдена ли метрика.
	GetGauge(ctx context.Context, name string) (float64, bool)

	// GetCounter возвращает текущее значение метрики типа Counter и найдена ли метрика.
	GetCounter(ctx context.Context, name string) (int64, bool)

	// Gauges возвращает карту всех сохраненных метрик типа Gauge.
	Gauges(ctx context.Context) map[string]float64

	// Counters возвращает карту всех сохраненных метрик типа Counter.
	Counters(ctx context.Context) map[string]int64

	// UpdateBatch выполняет батчевое обновление метрик.
	UpdateBatch(ctx context.Context, mr []model.Metrics) error
}

// Pinger определяет контракт для проверки работоспособности.
type Pinger interface {
	// Ping проверяет доступность сервиса.
	Ping(context.Context) error
}

// MetricService управляет бизнес-логикой работы с метриками.
type MetricService struct {
	repo   Repository
	logger *zap.Logger
	audit  Notifier
}

type metricRequest struct {
	typ   string
	name  string
	value string
}

// Notifier определяет контракт для отправки событий аудита.
type Notifier interface {
	// Notify отправляет событие в систему аудита.
	Notify(event audit.Event)
}

// NewMetricService создает новый экземпляр MetricService.
func NewMetricService(repo Repository, logger *zap.Logger, audit Notifier) MetricService {
	return MetricService{
		repo:   repo,
		logger: logger,
		audit:  audit,
	}
}
