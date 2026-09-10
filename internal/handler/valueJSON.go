package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/BREJJNEVV/metrics/internal/model"
)

func (ms *MetricService) GetValueJSON(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "wrong Content-Type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var request model.Metrics

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if request.ID == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	switch request.MType {
	case model.Gauge:
		val, found := ms.repo.GetGauge(r.Context(), request.ID)
		if !found {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		request.Value = &val

	case model.Counter:
		val, found := ms.repo.GetCounter(r.Context(), request.ID)
		if !found {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		request.Delta = &val

	default:
		http.Error(w, "unexpected type of metric", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(request)
}
