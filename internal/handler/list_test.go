package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
)

func TestListMetrics(t *testing.T) {

	tests := []struct {
		name        string
		method      string
		mockGauge   map[string]float64
		mockCounter map[string]int64
		wantCT      string
		wantCode    int
		wantBody    string
	}{
		{
			name:        "all ok",
			method:      http.MethodGet,
			mockGauge:   map[string]float64{"Alloc": 123.45},
			mockCounter: nil,
			wantCode:    http.StatusOK,
			wantCT:      "text/html; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRepository{
				gauge:   tt.mockGauge,
				counter: tt.mockCounter,
			}
			handler := MetricService{
				repo: mock,
			}

			r := chi.NewRouter()
			r.Get("/", handler.ListMetrics)

			req := httptest.NewRequest(tt.method, "/", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, tt.wantCT) {
				t.Errorf("Content-Type should contain text/html, got %q", ct)
			}

			if w.Code != tt.wantCode {
				t.Errorf(" expect statusCode %v, got %v", tt.wantCode, w.Code)
			}
			bd := w.Body.String()
			if !strings.Contains(bd, "Alloc = 123.45") {
				t.Error("body should contain 'Alloc = 123.45'")
			}
		})
	}
}

func (mr *mockRepository) Gauges() map[string]float64 {
	return mr.gauge
}

func (mr *mockRepository) Counters() map[string]int64 {
	return mr.counter
}
