package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestGetValueJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		ct          string
		mockGauge   map[string]float64
		mockCounter map[string]int64
		wantCode    int
		wantBody    string
	}{
		{
			name:      "gauge ok",
			body:      `{"id":"Alloc","type":"gauge"}`,
			ct:        "application/json",
			mockGauge: map[string]float64{"Alloc": 123.45},
			wantCode:  http.StatusOK,
			wantBody:  `"value":123.45`,
		},
		{
			name:        "counter ok",
			body:        `{"id":"PollCount","type":"counter"}`,
			ct:          "application/json",
			mockCounter: map[string]int64{"PollCount": 10},
			wantCode:    http.StatusOK,
			wantBody:    `"delta":10`,
		},
		{
			name:     "not found",
			body:     `{"id":"Unknown","type":"gauge"}`,
			ct:       "application/json",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "bad content-type",
			body:     `{"id":"Alloc","type":"gauge"}`,
			ct:       "text/plain",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "bad json",
			body:     `{`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty id",
			body:     `{"id":"","type":"gauge"}`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "unknown type",
			body:     `{"id":"X","type":"weird"}`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRepository{
				gauge:   tt.mockGauge,
				counter: tt.mockCounter,
			}
			service := MetricService{repo: mock, logger: zap.NewNop()}

			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.ct)
			w := httptest.NewRecorder()
			service.GetValueJSON(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status: got %d, want %d (body=%s)", w.Code, tt.wantCode, w.Body.String())
			}
			if tt.wantCode == http.StatusOK {
				if !strings.Contains(w.Body.String(), tt.wantBody) {
					t.Errorf("body: got %s, want substring %s", w.Body.String(), tt.wantBody)
				}
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type: got %s", w.Header().Get("Content-Type"))
				}
			}
		})
	}
}
