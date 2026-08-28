package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/rand"
	"net/http"
	"runtime"
	"sync"

	"github.com/BREJJNEVV/metrics/internal/compress"
	"github.com/BREJJNEVV/metrics/internal/model"
	"go.uber.org/zap"
)

type MetricsWriter interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
}

type MetricsReader interface {
	Gauges() map[string]float64
	Counters() map[string]int64
}

type MetricsStorage struct {
	counter map[string]int64
	gauge   map[string]float64
	mu      sync.Mutex
}

func Collect(mw MetricsWriter) {
	allMetrics := runtime.MemStats{}
	runtime.ReadMemStats(&allMetrics)

	mw.SetGauge("Alloc", float64(allMetrics.Alloc))
	mw.SetGauge("BuckHashSys", float64(allMetrics.BuckHashSys))
	mw.SetGauge("Frees", float64(allMetrics.Frees))
	mw.SetGauge("GCCPUFraction", allMetrics.GCCPUFraction)
	mw.SetGauge("GCSys", float64(allMetrics.GCSys))
	mw.SetGauge("HeapAlloc", float64(allMetrics.HeapAlloc))
	mw.SetGauge("HeapIdle", float64(allMetrics.HeapIdle))
	mw.SetGauge("HeapInuse", float64(allMetrics.HeapInuse))
	mw.SetGauge("HeapObjects", float64(allMetrics.HeapObjects))
	mw.SetGauge("HeapReleased", float64(allMetrics.HeapReleased))
	mw.SetGauge("HeapSys", float64(allMetrics.HeapSys))
	mw.SetGauge("LastGC", float64(allMetrics.LastGC))
	mw.SetGauge("Lookups", float64(allMetrics.Lookups))
	mw.SetGauge("MCacheInuse", float64(allMetrics.MCacheInuse))
	mw.SetGauge("MCacheSys", float64(allMetrics.MCacheSys))
	mw.SetGauge("MSpanInuse", float64(allMetrics.MSpanInuse))
	mw.SetGauge("MSpanSys", float64(allMetrics.MSpanSys))
	mw.SetGauge("Mallocs", float64(allMetrics.Mallocs))
	mw.SetGauge("NextGC", float64(allMetrics.NextGC))
	mw.SetGauge("NumForcedGC", float64(allMetrics.NumForcedGC))
	mw.SetGauge("NumGC", float64(allMetrics.NumGC))
	mw.SetGauge("OtherSys", float64(allMetrics.OtherSys))
	mw.SetGauge("PauseTotalNs", float64(allMetrics.PauseTotalNs))
	mw.SetGauge("StackInuse", float64(allMetrics.StackInuse))
	mw.SetGauge("StackSys", float64(allMetrics.StackSys))
	mw.SetGauge("Sys", float64(allMetrics.Sys))
	mw.SetGauge("TotalAlloc", float64(allMetrics.TotalAlloc))

	mw.AddCounter("PollCount", 1)
	mw.SetGauge("RandomValue", rand.Float64())
}

func Send(mr MetricsReader, client *http.Client, baseURL string, logger *zap.Logger) {
	for name, value := range mr.Counters() {
		var metric model.Metrics
		metric.ID = name
		metric.MType = model.Counter
		metric.Delta = &value
		data, err := json.Marshal(metric)
		if err != nil {
			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}
		comressData, err := compress.Compress(data)
		if err != nil {
			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}

		url := fmt.Sprintf("%s/update", baseURL)
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(comressData))
		if err != nil {
			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Encoding", "gzip")

		resp, err := client.Do(request)
		if err != nil {
			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logger.Warn("unexpected status code", zap.String("name", name), zap.Int("status", resp.StatusCode))
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	for name, value := range mr.Gauges() {
		var metric model.Metrics
		metric.ID = name
		metric.MType = model.Gauge
		metric.Value = &value
		data, err := json.Marshal(metric)
		if err != nil {
			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}

		comressData, err := compress.Compress(data)
		if err != nil {
			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}

		url := fmt.Sprintf("%s/update", baseURL)
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(comressData))
		if err != nil {
			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}

		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Encoding", "gzip")

		resp, err := client.Do(request)
		if err != nil {

			logger.Error("error sending", zap.String("name", name), zap.Error(err))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			logger.Warn("unexpected status code", zap.String("name", name), zap.Int("status", resp.StatusCode))
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func CreateMetricStorage() *MetricsStorage {
	return &MetricsStorage{
		counter: make(map[string]int64),
		gauge:   make(map[string]float64),
	}
}

func (ms *MetricsStorage) SetGauge(name string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauge[name] = value
}

func (ms *MetricsStorage) AddCounter(name string, value int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counter[name] += value
}

func (ms *MetricsStorage) Gauges() map[string]float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	copyMap := make(map[string]float64, len(ms.gauge))
	maps.Copy(copyMap, ms.gauge)
	return copyMap
}

func (ms *MetricsStorage) Counters() map[string]int64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	copyMap := make(map[string]int64, len(ms.counter))
	maps.Copy(copyMap, ms.counter)
	return copyMap
}

func (ms *MetricsStorage) ResetCounter(name string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counter[name] = 0
}
