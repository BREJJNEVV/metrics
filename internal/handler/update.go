package handler

import (
	"net"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/BREJJNEVV/metrics/internal/audit"
	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

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
		err = ms.repo.Set(r.Context(), request.name, fGauge)
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
		err = ms.repo.Add(r.Context(), request.name, icounter)
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

	ms.audit.Notify(audit.Event{
		TS:      time.Now().Unix(),
		Metrics: []string{request.name},
		IPAddr:  clientIP(r),
	})

	w.WriteHeader(http.StatusOK)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
