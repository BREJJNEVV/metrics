// Package memory определяет интерфейс хранилища метрик в памяти.
package memory

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/BREJJNEVV/metrics/internal/model"
)

// MemStorage — это потокобезопасное хранилище метрик в памяти.
type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	mu      sync.Mutex
}

// Create создает и инициализирует новое пустое хранилище.
func Create() *MemStorage {
	return &MemStorage{
		counter: make(map[string]int64),
		gauge:   make(map[string]float64),
	}
}

// Set потокобезопасно устанавливает значение метрики типа gauge.
func (ms *MemStorage) Set(ctx context.Context, name string, value float64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauge[name] = value
	return nil
}

// Add потокобезопасно увеличивает значение метрики типа counter.
func (ms *MemStorage) Add(ctx context.Context, name string, value int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counter[name] += value
	return nil
}

// GetGauge возвращает значение метрики gauge.
// Возвращает false, если метрика не найдена.
func (ms *MemStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	v, ok := ms.gauge[name]
	if !ok {
		return 0, false
	}
	return v, true
}

// GetCounter возвращает значение метрики counter.
// Возвращает false, если метрика не найдена.
func (ms *MemStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	v, ok := ms.counter[name]
	if !ok {
		return 0, false
	}
	return v, true
}

// Gauges возвращает копию всех метрик типа gauge.
func (ms *MemStorage) Gauges(ctx context.Context) map[string]float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	newMap := make(map[string]float64, len(ms.gauge))
	maps.Copy(newMap, ms.gauge)

	return newMap
}

// Counters возвращает копию всех метрик типа counter.
func (ms *MemStorage) Counters(ctx context.Context) map[string]int64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	newMap := make(map[string]int64, len(ms.counter))
	maps.Copy(newMap, ms.counter)

	return newMap
}

// UpdateBatch потокобезопасно обновляет батч метрик.
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

// Ping реализует интерфейс Pinger.
func (ms *MemStorage) Ping(ctx context.Context) error {
	return nil
}
