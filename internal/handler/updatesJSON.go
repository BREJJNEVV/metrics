package handler

import (
	"encoding/json"
	"strings"

	"net/http"

	"github.com/BREJJNEVV/metrics/internal/model"
	"go.uber.org/zap"
)

func (ms *MetricService) UpdatesJSON(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "wrong Content-Type", http.StatusBadRequest)
		return
	}
	var requests []model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, request := range requests {
		if request.ID == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		switch request.MType {
		case model.Gauge:
			if request.Value == nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		case model.Counter:
			if request.Delta == nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	if err := ms.repo.UpdateBatch(requests); err != nil {
		ms.logger.Error("batch update failed",
			zap.Error(err),
			zap.Int("metrics_count", len(requests)),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
