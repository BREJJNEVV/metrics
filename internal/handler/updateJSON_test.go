package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestUpdateJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		ct         string
		wantCode   int
		wantSet    bool
		wantAdd    bool
		wantMetric string
	}{
		{
			name:       "gauge ok",
			body:       `{"id":"Alloc","type":"gauge","value":123.45}`,
			ct:         "application/json",
			wantCode:   http.StatusOK,
			wantSet:    true,
			wantMetric: "Alloc",
		},
		{
			name:       "counter ok",
			body:       `{"id":"PollCount","type":"counter","delta":10}`,
			ct:         "application/json",
			wantCode:   http.StatusOK,
			wantAdd:    true,
			wantMetric: "PollCount",
		},
		{
			name:     "wrong content-type",
			body:     `{"id":"Alloc","type":"gauge","value":1}`,
			ct:       "text/plain",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "bad json",
			body:     `{not json`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty id",
			body:     `{"id":"","type":"gauge","value":1}`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "gauge without value",
			body:     `{"id":"Alloc","type":"gauge"}`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "counter without delta",
			body:     `{"id":"PollCount","type":"counter"}`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "unknown type",
			body:     `{"id":"X","type":"weird","value":1}`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRepository{}
			notifier := &mockNotifier{}
			service := MetricService{repo: mock, logger: zap.NewNop(), audit: notifier}

			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.ct)
			w := httptest.NewRecorder()
			service.UpdateJSON(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status: got %d, want %d (body=%s)", w.Code, tt.wantCode, w.Body.String())
			}
			if mock.SetCall != tt.wantSet {
				t.Errorf("SetCall: got %v, want %v", mock.SetCall, tt.wantSet)
			}
			if mock.AddCall != tt.wantAdd {
				t.Errorf("AddCall: got %v, want %v", mock.AddCall, tt.wantAdd)
			}

			if tt.wantCode == http.StatusOK {
				if len(notifier.events) != 1 {
					t.Fatalf("audit events: got %d, want 1", len(notifier.events))
				}
				if notifier.events[0].Metrics[0] != tt.wantMetric {
					t.Errorf("audit metric: got %v, want %s", notifier.events[0].Metrics, tt.wantMetric)
				}
				if notifier.events[0].TS <= 0 {
					t.Error("audit TS should be positive")
				}
			} else {
				if len(notifier.events) != 0 {
					t.Errorf("audit should not fire on error, got %d events", len(notifier.events))
				}
			}
		})
	}
}
