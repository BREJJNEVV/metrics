package handler

import (
	"context"

	"github.com/BREJJNEVV/metrics/internal/audit"
	"github.com/BREJJNEVV/metrics/internal/model"
	"go.uber.org/zap"
)

type Repository interface {
	Set(ctx context.Context, name string, value float64) error
	Add(ctx context.Context, name string, value int64) error
	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	Gauges(ctx context.Context) map[string]float64
	Counters(ctx context.Context) map[string]int64
	UpdateBatch(ctx context.Context, mr []model.Metrics) error
}

type Pinger interface {
	Ping(context.Context) error
}

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

type Notifier interface {
	Notify(event audit.Event)
}

func CreateMetricService(repo Repository, logger *zap.Logger, audit Notifier) MetricService {
	return MetricService{
		repo:   repo,
		logger: logger,
		audit:  audit,
	}
}
