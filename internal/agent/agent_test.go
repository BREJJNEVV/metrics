package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
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

			requests = append(requests, requestLog{
				Method:      r.Method,
				Path:        r.URL.Path,
				ContentType: r.Header.Get("Content-Type"),
			})
			w.WriteHeader(http.StatusOK)
		},
	))

	defer server.Close()

	Send(mr, client, server.URL)

	if len(requests) != 2 {
		t.Fatal("requests has not be send")
	}

	expectedResponse := map[string]bool{
		"/update/counter/PollCount/10": false,
		"/update/gauge/Alloc/123.45":   false,
	}

	for _, v := range requests {
		if v.ContentType != "text/plain" {
			t.Errorf("context is not right: %v", v.ContentType)
		}
		if v.Method != http.MethodPost {
			t.Errorf("method is not post: %v", v.Method)
		}
		_, ok := expectedResponse[v.Path]
		if !ok {
			t.Fatalf("value not found: %v", v.Path)
		}
	}
}
