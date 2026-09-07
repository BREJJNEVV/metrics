package persistence

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/BREJJNEVV/metrics/internal/repository/memory"
)

func NewStorage(restore bool, path string) (*memory.MemStorage, error) {
	ms := memory.Create()
	if restore {
		err := LoadMetrics(ms, path)
		if err != nil {
			return nil, err
		}
	}
	return ms, nil
}

type SyncSaver struct {
	*memory.MemStorage
	path string
}

func CreateSyncSaver(ms *memory.MemStorage, path string) *SyncSaver {
	return &SyncSaver{
		MemStorage: ms,
		path:       path,
	}
}

func (ss *SyncSaver) Set(name string, value float64) error {
	err := ss.MemStorage.Set(name, value)
	if err != nil {
		return err
	}
	return SaveMetrics(ss.MemStorage, ss.path)
}

func (ss *SyncSaver) Add(name string, value int64) error {
	err := ss.MemStorage.Add(name, value)
	if err != nil {
		return err
	}
	return SaveMetrics(ss.MemStorage, ss.path)
}

func (ss *SyncSaver) UpdateBatch(mr []model.Metrics) error {
	if err := ss.MemStorage.UpdateBatch(mr); err != nil {
		return err
	}
	return nil
}

func SaveMetrics(m *memory.MemStorage, path string) error {
	metricSlice := make([]model.Metrics, 0, len(m.Gauges())+len(m.Counters()))

	for name, value := range m.Gauges() {
		v := value
		metricSlice = append(metricSlice, model.Metrics{
			ID:    name,
			Value: &v,
			MType: model.Gauge,
		})
	}

	for name, value := range m.Counters() {
		v := value
		metricSlice = append(metricSlice, model.Metrics{
			ID:    name,
			Delta: &v,
			MType: model.Counter,
		})
	}

	data, err := json.Marshal(metricSlice)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func LoadMetrics(m *memory.MemStorage, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}

	sliceMetric := []model.Metrics{}
	err = json.Unmarshal(data, &sliceMetric)
	if err != nil {
		return err
	}

	for _, s := range sliceMetric {
		switch s.MType {
		case model.Gauge:
			if s.Value != nil {
				m.Set(s.ID, *s.Value)
			}
		case model.Counter:
			if s.Delta != nil {
				m.Add(s.ID, *s.Delta)
			}
		}
	}
	return nil
}
