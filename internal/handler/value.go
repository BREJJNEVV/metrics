package handler

import (
	"fmt"
	"net/http"

	models "github.com/BREJJNEVV/metrics/internal/model"
	"github.com/go-chi/chi"
)

func (ms *MetricService) GetValue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")

	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	switch typ {
	case models.Gauge:
		val, found := ms.repo.GetGauge(name)
		if !found {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%g", val)

	case models.Counter:
		val, found := ms.repo.GetCounter(name)
		if !found {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%d", val)

	default:
		http.Error(w, "unexpected type of metric", http.StatusBadRequest)
		return
	}
}
