package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BREJJNEVV/metrics/internal/model"
	"go.uber.org/zap"
)

type mockWriter struct {
	gauges   map[string]float64
	counters map[string]int64
}

func (m *mockWriter) AddCounter(name string, value int64) {
	m.counters[name] += value
}
func (m *mockWriter) SetGauge(name string, value float64) {
	m.gauges[name] = value
}

type mockReader struct {
	gauges   map[string]float64
	counters map[string]int64
}

func (m *mockReader) Gauges() map[string]float64 {
	return m.gauges
}

func (m *mockReader) Counters() map[string]int64 {
	return m.counters
}

func CreateMockWriter() *mockWriter {
	return &mockWriter{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func CreateMockReader(gauges map[string]float64, counters map[string]int64) *mockReader {
	return &mockReader{
		gauges:   gauges,
		counters: counters,
	}
}
func TestCollect(t *testing.T) {
	mw := CreateMockWriter()
	Collect(mw)
	v1, ok := mw.gauges["Alloc"]
	if !ok {
		t.Fatal("Alloc not found in gauges")
	}
	if v1 <= 0 {
		t.Errorf("Alloc has unexpected value: %v", v1)
	}
	if len(mw.gauges) != 28 {
		t.Errorf("Not all merics recived, only %d", len(mw.gauges))
	}

	v2, ok := mw.counters["PollCount"]
	if !ok {
		t.Error("PollCount not found in counters")
		return
	}
	if v2 != 1 {
		t.Errorf("PollCount has unexpected value: %v", v2)
	}

}

type requestLog struct {
	Method      string
	Path        string
	ContentType string
	Body        []byte
}

func TestSend(t *testing.T) {

	client := &http.Client{}
	mr := CreateMockReader(
		map[string]float64{"Alloc": 123.45},
		map[string]int64{"PollCount": 10},
	)

	var requests []requestLog

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			body, _ := io.ReadAll(r.Body)
			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				gz, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					t.Errorf("failed to create gzip reader: %v", err)
					return
				}
				defer gz.Close()
				body, err = io.ReadAll(gz)
				if err != nil {
					t.Errorf("failed to decompress request body: %v", err)
					return
				}
			}

			requests = append(requests, requestLog{
				Method:      r.Method,
				Path:        r.URL.Path,
				ContentType: r.Header.Get("Content-Type"),
				Body:        body,
			})
			w.WriteHeader(http.StatusOK)
		},
	))

	defer server.Close()
	logger, _ := zap.NewDevelopment()
	Send(mr, client, server.URL, logger)

	if len(requests) != 1 {
		t.Fatalf("expected 1 requests, got %d", len(requests))
	}

	for _, req := range requests {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", req.Method)
		}
		if req.Path != "/updates" {
			t.Errorf("expected path /updates, got %s", req.Path)
		}
		if req.ContentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", req.ContentType)
		}

		// var metric model.Metrics
		var mericsSlice []model.Metrics
		if err := json.Unmarshal(req.Body, &mericsSlice); err != nil {
			t.Errorf("failed to unmarshal request body: %v", err)
			continue
		}
		for _, metric := range mericsSlice {
			switch metric.MType {
			case model.Gauge:
				if metric.ID != "Alloc" {
					t.Errorf("expected gauge id Alloc, got %s", metric.ID)
				}
				if metric.Value == nil || *metric.Value != 123.45 {
					t.Errorf("expected gauge value 123.45, got %v", metric.Value)
				}
			case model.Counter:
				if metric.ID != "PollCount" {
					t.Errorf("expected counter id PollCount, got %s", metric.ID)
				}
				if metric.Delta == nil || *metric.Delta != 10 {
					t.Errorf("expected counter delta 10, got %v", metric.Delta)
				}
			default:
				t.Errorf("unexpected metric type: %s", metric.MType)
			}
		}

	}
}
