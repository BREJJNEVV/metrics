package handler

import (
	"strconv"
	"strings"

	"net/http"

	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type Repository interface {
	Set(name string, value float64) error
	Add(name string, value int64) error
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	Gauges() map[string]float64
	Counters() map[string]int64
	UpdateBatch(mr []model.Metrics) error
}

type MetricService struct {
	repo   Repository
	logger *zap.Logger
}

type metricRequest struct {
	typ   string
	name  string
	value string
}

func (ms *MetricService) Update(w http.ResponseWriter, r *http.Request) {
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
		err = ms.repo.Set(request.name, fGauge)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			ms.logger.Error("Update metric error",
				zap.String("type", request.typ),
				zap.String("name", request.name),
				zap.Error(err),
			)
			return
		}
	case model.Counter:
		icounter, err := strconv.ParseInt(request.value, 10, 64)
		if err != nil {
			http.Error(w, "status bad request", http.StatusBadRequest)
			return
		}
		err = ms.repo.Add(request.name, icounter)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			ms.logger.Error("Update metric error",
				zap.String("type", request.typ),
				zap.String("name", request.name),
				zap.Error(err),
			)
			return
		}
	default:
		http.Error(w, "status bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func CreateMetricService(repo Repository, logger *zap.Logger) MetricService {
	return MetricService{
		repo:   repo,
		logger: logger,
	}
}
