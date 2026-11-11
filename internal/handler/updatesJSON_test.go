package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestUpdatesJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		ct          string
		wantCode    int
		wantBatch   bool
		wantMetrics []string
	}{
		{
			name:        "batch ok",
			body:        `[{"id":"Alloc","type":"gauge","value":1},{"id":"PollCount","type":"counter","delta":5}]`,
			ct:          "application/json",
			wantCode:    http.StatusOK,
			wantBatch:   true,
			wantMetrics: []string{"Alloc", "PollCount"},
		},
		{
			name:     "wrong content-type",
			body:     `[]`,
			ct:       "text/plain",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "bad json",
			body:     `[`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "one invalid gauge",
			body:     `[{"id":"Alloc","type":"gauge","value":1},{"id":"X","type":"gauge"}]`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "unknown type in batch",
			body:     `[{"id":"A","type":"weird","value":1}]`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty id in batch",
			body:     `[{"id":"","type":"gauge","value":1}]`,
			ct:       "application/json",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRepository{}
			notifier := &mockNotifier{}
			service := MetricService{repo: mock, logger: zap.NewNop(), audit: notifier}

			req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.ct)
			w := httptest.NewRecorder()
			service.UpdatesJSON(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status: got %d, want %d (body=%s)", w.Code, tt.wantCode, w.Body.String())
			}
			if mock.BatchCalled != tt.wantBatch {
				t.Errorf("BatchCalled: got %v, want %v", mock.BatchCalled, tt.wantBatch)
			}

			if tt.wantCode == http.StatusOK {
				if len(notifier.events) != 1 {
					t.Fatalf("audit events: got %d, want 1", len(notifier.events))
				}
				got := notifier.events[0].Metrics
				if len(got) != len(tt.wantMetrics) {
					t.Fatalf("metrics count: got %v, want %v", got, tt.wantMetrics)
				}
				for i := range got {
					if got[i] != tt.wantMetrics[i] {
						t.Errorf("metric[%d]: got %s, want %s", i, got[i], tt.wantMetrics[i])
					}
				}
			} else if len(notifier.events) != 0 {
				t.Errorf("audit should not fire on error")
			}
		})
	}
}
