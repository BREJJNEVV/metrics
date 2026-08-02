package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
)

func TestGetValue(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		mockGauge   map[string]float64
		mockCounter map[string]int64
		wantCode    int
		wantBody    string
		wantCT      string
	}{
		{
			name:        "get gauge ok",
			method:      http.MethodGet,
			url:         "/value/gauge/Alloc",
			mockGauge:   map[string]float64{"Alloc": 123.45},
			mockCounter: nil,
			wantCode:    http.StatusOK,
			wantBody:    "123.45",
			wantCT:      "text/plain",
		},
		{
			name:        "get counter ok",
			method:      http.MethodGet,
			url:         "/value/counter/PollCount",
			mockGauge:   nil,
			mockCounter: map[string]int64{"PollCount": 10},
			wantCode:    http.StatusOK,
			wantBody:    "10",
			wantCT:      "text/plain",
		},
		{
			name:        "not found",
			method:      http.MethodGet,
			url:         "/value/gauge/NotFound",
			mockGauge:   map[string]float64{},
			mockCounter: nil,
			wantCode:    http.StatusNotFound,
			wantBody:    "metric not found\n",
			wantCT:      "text/plain; charset=utf-8",
		},
		{
			name:        "bad type",
			method:      http.MethodGet,
			url:         "/value/xyz/some",
			mockGauge:   nil,
			mockCounter: nil,
			wantCode:    http.StatusBadRequest,
			wantBody:    "unexpected type of metric\n",
			wantCT:      "text/plain; charset=utf-8",
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
			r.Get("/value/{type}/{name}", handler.GetValue)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Header().Get("Content-Type") != tt.wantCT {
				t.Errorf("expect Content-Type %s, got %s", tt.wantCT, w.Header().Get("Content-Type"))
			}
			if w.Code != tt.wantCode {
				t.Errorf(" expect statusCode %v, got %v", tt.wantCode, w.Code)
			}
			if w.Body.String() != tt.wantBody {
				t.Errorf("exp value %s, got %s", tt.wantBody, w.Body.String())
			}

		})

	}

}

func (mr *mockRepository) GetGauge(name string) (float64, bool) {
	v, ok := mr.gauge[name]
	if !ok {
		return 0, false
	}
	return v, true
}

func (mr *mockRepository) GetCounter(name string) (int64, bool) {
	v, ok := mr.counter[name]
	if !ok {
		return 0, false
	}
	return v, true
}
