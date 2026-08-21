package handler

import (
	"encoding/json"
	"log"
	"strings"

	"net/http"

	"github.com/BREJJNEVV/metrics/internal/model"
)

func (h *MetricService) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "wrong Content-Type", http.StatusBadRequest)
		return
	}
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
		if request.Value == nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		err = h.repo.Set(request.ID, *request.Value)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Printf("request type: %s, request name: %s, error: %v. ", request.MType, request.ID, err)
			return
		}
	case model.Counter:
		if request.Delta == nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		err = h.repo.Add(request.ID, *request.Delta)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Printf("request type: %s, request name: %s, error: %v. ", request.MType, request.ID, err)
			return
		}
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
