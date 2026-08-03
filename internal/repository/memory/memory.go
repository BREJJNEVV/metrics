package memory

import (
	"maps"
	"sync"
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

func (ms *MemStorage) Set(name string, value float64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauge[name] = value
	return nil
}

func (ms *MemStorage) Add(name string, value int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counter[name] += value
	return nil
}

func (ms *MemStorage) GetGauge(name string) (float64, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	v, ok := ms.gauge[name]
	if !ok {
		return 0, false
	}
	return v, true
}

func (ms *MemStorage) GetCounter(name string) (int64, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	v, ok := ms.counter[name]
	if !ok {
		return 0, false
	}
	return v, true
}

func (ms *MemStorage) Gauges() map[string]float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	newMap := make(map[string]float64, len(ms.gauge))
	maps.Copy(newMap, ms.gauge)

	return newMap

}

func (ms *MemStorage) Counters() map[string]int64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	newMap := make(map[string]int64, len(ms.counter))
	maps.Copy(newMap, ms.counter)

	return newMap

}
