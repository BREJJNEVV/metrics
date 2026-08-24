package handler

import (
	"log"
	"strconv"
	"strings"

	"net/http"

	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/BREJJNEVV/metrics/internal/repository/memory"
	"github.com/go-chi/chi"
)

type Repository interface {
	Set(name string, value float64) error
	Add(name string, value int64) error
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	Gauges() map[string]float64
	Counters() map[string]int64
}

type MetricService struct {
	repo Repository
}

type metricRequest struct {
	typ   string
	name  string
	value string
}

func (h *MetricService) Update(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "text/plain") {
		http.Error(w, "wrong Content-Type", http.StatusBadRequest)
		return
	}

	request := metricRequest{
		typ:   chi.URLParam(r, "type"),
		name:  chi.URLParam(r, "name"),
		value: chi.URLParam(r, "value"),
	}

	if request.name == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch request.typ {
	case model.Gauge:
		fGauge, err := strconv.ParseFloat(request.value, 64)
		if err != nil {
			http.Error(w, "status bad request", http.StatusBadRequest)
			return
		}
		err = h.repo.Set(request.name, fGauge)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Printf("request type: %s, request name: %s, error: %v. ", request.typ, request.name, err)
			return
		}
	case model.Counter:
		icounter, err := strconv.ParseInt(request.value, 10, 64)
		if err != nil {
			http.Error(w, "status bad request", http.StatusBadRequest)
			return
		}
		err = h.repo.Add(request.name, icounter)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Printf("request type: %s, request name: %s, error: %v. ", request.typ, request.name, err)
			return
		}
	default:
		http.Error(w, "status bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func CreateMetricService() MetricService {
	return MetricService{
		repo: memory.Create(),
	}
}
