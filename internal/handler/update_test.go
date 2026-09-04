package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
)

type mockRepository struct {
	AddCall  bool
	AddName  string
	AddValue int64
	SetCall  bool
	SetName  string
	SetValue float64
	GetCall  bool

	gauge   map[string]float64
	counter map[string]int64
}

func TestUpdateGauge(t *testing.T) {

	mock := &mockRepository{}
	updHandler := MetricService{
		repo: mock,
	}

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	req.Header.Set("Content-Type", "text/plain")

	r := chi.NewRouter()
	r.Post("/update/{type:.*}/{name:.*}/{value:.*}", updHandler.Update)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !mock.SetCall {
		t.Error("request is not saved")
	}
	if mock.SetName != "Alloc" {
		t.Errorf("Recive wrong metric name, exp: %s, find: %s", "Alloc", mock.SetName)
	}
	if mock.SetValue != 123.45 {
		t.Errorf("Recive wrong metric Alloc value, exp: %g, find: %g", 123.45, mock.SetValue)
	}
	if mock.AddCall {
		t.Error("Add should not be called ")
	}

}

func TestUpdateCounter(t *testing.T) {

	mock := &mockRepository{}
	updHandler := MetricService{
		repo: mock,
	}

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/10", nil)
	req.Header.Set("Content-Type", "text/plain")

	r := chi.NewRouter()
	r.Post("/update/{type:.*}/{name:.*}/{value:.*}", updHandler.Update)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !mock.AddCall {
		t.Error("request is not saved")
	}
	if mock.AddName != "PollCount" {
		t.Errorf("Recive wrong metric name, exp: %s, find: %s", "PollCount", mock.AddName)
	}
	if mock.AddValue != 10 {
		t.Errorf("Recive wrong metric Alloc value, exp: %d, find: %d", 10, mock.AddValue)
	}
	if mock.SetCall {
		t.Error("Set should not be called ")
	}

}

func TestUpdateErrors(t *testing.T) {

	tests := []struct {
		name        string
		method      string
		url         string
		contentType string
		code        int
	}{
		{
			name:        "missing type of metric",
			method:      http.MethodPost,
			url:         "/update//counter/1",
			contentType: "text/plain",
			code:        http.StatusBadRequest,
		},
		{
			name:        "missing name",
			method:      http.MethodPost,
			url:         "/update/gauge//1.23",
			contentType: "text/plain",
			code:        http.StatusNotFound,
		},
		{
			name:        "missing value",
			method:      http.MethodPost,
			url:         "/update/gauge/Alloc/",
			contentType: "text/plain",
			code:        http.StatusNotFound,
		},
		{
			name:        "wrong type of metric",
			method:      http.MethodPost,
			url:         "/update/raffuge/Alloc/12.34",
			contentType: "text/plain",
			code:        http.StatusBadRequest,
		},
		{
			name:        "wrong method",
			method:      http.MethodGet,
			url:         "/update/gauge/Alloc/12.34",
			contentType: "text/plain",
			code:        http.StatusMethodNotAllowed,
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			url:         "/update/gauge/Alloc/12.34",
			contentType: "application/json",
			code:        http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRepository{}
			service := MetricService{repo: mock}
			req := httptest.NewRequest(tt.method, tt.url, nil)

			r := chi.NewRouter()
			r.Post("/update/{type:.*}/{name:.*}/{value:.*}", service.Update)
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.code {
				t.Errorf("expected status %d, got %d", w.Code, tt.code)
			}
			if mock.AddCall || mock.SetCall {
				t.Error("no repo methods should be called")
			}

		})
	}
}

func (mr *mockRepository) Set(name string, value float64) error {
	mr.SetCall = true
	mr.SetName = name
	mr.SetValue = value
	return nil
}

func (mr *mockRepository) Add(name string, value int64) error {
	mr.AddCall = true
	mr.AddName = name
	mr.AddValue = value
	return nil
}
