package handler

import (
	"log"
	"strconv"
	"strings"

	"net/http"

	models "github.com/BREJJNEVV/metrics/internal/model"
	"github.com/BREJJNEVV/metrics/internal/repository/db"
	"github.com/go-chi/chi"
)

type Repository interface {
	Set(name string, value float64) error // для gauge
	Add(name string, value int64) error   // для counter
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

// Сервер сохраняет метрики, передаваемые ему от агента в базе данных
func (h *MetricService) Update(w http.ResponseWriter, r *http.Request) {
	//directory := "update.handlerUpdate"

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
	// log.Print(parts)
	log.Printf("value bytes: %v, string: %q", []byte(request.value), request.value)
	log.Printf("typ=%q, name=%q, value=%q", request.typ, request.name, request.value)
	switch request.typ {
	case models.Gauge:
		fGauge, err := strconv.ParseFloat(request.value, 64)
		if err != nil {
			http.Error(w, "status bad request", http.StatusBadRequest)
			return
		}
		err = h.repo.Set(request.name, fGauge)
		if err != nil {
			http.Error(w, "InternalServerError", http.StatusInternalServerError)
			return
		}
	case models.Counter:
		icounter, err := strconv.ParseInt(request.value, 10, 64)
		if err != nil {
			http.Error(w, "status bad request2", http.StatusBadRequest)
			return
		}
		err = h.repo.Add(request.name, icounter)
		if err != nil {
			http.Error(w, "InternalServerError", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "status bad request3", http.StatusBadRequest)
		return
	}

	//w.Header().Set("Content-Type", "application/json")
	//w.Write([]byte(`{"status": "ok"}`))
	w.WriteHeader(http.StatusOK)
	log.Println("Successfully handled, sent 200")
}

func CreateMetricService() MetricService {
	return MetricService{
		repo: db.Create(),
	}
}
