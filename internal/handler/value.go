package handler

import (
	"fmt"
	"net/http"

	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/go-chi/chi"
)

func (ms *MetricService) GetValue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	switch typ {
	case model.Gauge:
		val, found := ms.repo.GetGauge(name)
		if !found {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%g", val)

	case model.Counter:
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
