package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

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

func ptr(v float64) *float64 { return &v }
func ptrInt(v int64) *int64  { return &v }

func TestSendBatch(t *testing.T) {

	client := &http.Client{}
	var requests []requestLog

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := ""
			if key != "" && r.Header.Get("HashSHA256") == "" {
				t.Error("missing HashSHA256 header")
			}
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
	ctx := context.Background()
	cfg := New(ctx, client, server.URL, "", logger, 1)
	batch := []model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: ptr(123.45)},
		{ID: "PollCount", MType: model.Counter, Delta: ptrInt(10)},
	}
	cfg.SendBatch(ctx, batch)

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

func TestCollectBatch(t *testing.T) {
	mr := CreateMockReader(
		map[string]float64{"Alloc": 100, "Sys": 200},
		map[string]int64{"PollCount": 5},
	)

	batch := CollectBatch(mr)

	if len(batch) != 3 {
		t.Fatalf("expected 3 metrics, got %d", len(batch))
	}

	byID := make(map[string]model.Metrics)
	for _, m := range batch {
		byID[m.ID] = m
	}

	if g, ok := byID["Alloc"]; !ok || g.Value == nil || *g.Value != 100 {
		t.Errorf("Alloc missing or wrong: %+v", g)
	}
	if g, ok := byID["Sys"]; !ok || g.Value == nil || *g.Value != 200 {
		t.Errorf("Sys missing or wrong: %+v", g)
	}
	if c, ok := byID["PollCount"]; !ok || c.Delta == nil || *c.Delta != 5 {
		t.Errorf("PollCount missing or wrong: %+v", c)
	}
}

func TestCollectBatch_Empty(t *testing.T) {
	mr := CreateMockReader(nil, nil)
	batch := CollectBatch(mr)
	if len(batch) != 0 {
		t.Errorf("expected empty batch, got %d", len(batch))
	}
}

func TestIsRetriable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"generic", errors.New("some error"), false},
		{"conn refused", syscall.ECONNREFUSED, true},
		{"conn reset", syscall.ECONNRESET, true},
		{"wrapped conn refused", fmt.Errorf("dial: %w", syscall.ECONNREFUSED), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetriable(tt.err); got != tt.want {
				t.Errorf("isRetriable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestSubmit(t *testing.T) {
	cfg := New(context.Background(), &http.Client{}, "http://localhost", "", zap.NewNop(), 1)

	batch := []model.Metrics{
		{ID: "A", MType: model.Gauge},
	}

	cfg.Submit(batch)

	select {
	case got := <-cfg.jobs:
		if len(got) != 1 || got[0].ID != "A" {
			t.Errorf("unexpected batch: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("Submit did not put task in channel")
	}
}
