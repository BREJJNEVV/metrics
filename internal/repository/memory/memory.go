package memory

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/BREJJNEVV/metrics/internal/model"
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	mu      sync.Mutex
}

func Create() *MemStorage {
	return &MemStorage{
		counter: make(map[string]int64),
		gauge:   make(map[string]float64),
	}
}

func (ms *MemStorage) Set(ctx context.Context, name string, value float64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauge[name] = value
	return nil
}

func (ms *MemStorage) Add(ctx context.Context, name string, value int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counter[name] += value
	return nil
}

func (ms *MemStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	v, ok := ms.gauge[name]
	if !ok {
		return 0, false
	}
	return v, true
}

func (ms *MemStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	v, ok := ms.counter[name]
	if !ok {
		return 0, false
	}
	return v, true
}

func (ms *MemStorage) Gauges(ctx context.Context) map[string]float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	newMap := make(map[string]float64, len(ms.gauge))
	maps.Copy(newMap, ms.gauge)

	return newMap

}

func (ms *MemStorage) Counters(ctx context.Context) map[string]int64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	newMap := make(map[string]int64, len(ms.counter))
	maps.Copy(newMap, ms.counter)

	return newMap
}

func (ms *MemStorage) UpdateBatch(ctx context.Context, mr []model.Metrics) error {
	var err error
	for _, request := range mr {
		switch request.MType {
		case model.Gauge:
			err = ms.Set(ctx, request.ID, *request.Value)
			if err != nil {
				return err
			}
		case model.Counter:
			err = ms.Add(ctx, request.ID, *request.Delta)
			if err != nil {
				return err
			}
		default:
			return errors.New("unknown metric type")
		}
	}
	return nil
}

func (p *MemStorage) Ping(ctx context.Context) error {
	return nil
}
