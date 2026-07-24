package update

import (
	"log"
	"strconv"

	"net/http"
	"strings"

	"github.com/BREJJNEVV/metrics/internal/repository/db"
)

type Repository interface {
	Set(name string, value float64) error // для gauge
	Add(name string, value int64) error   // для counter
}

type UpdateHandler struct {
	repo Repository
}

type metricRequest struct {
	typ   string
	name  string
	value string
}

func (h *UpdateHandler) Update(w http.ResponseWriter, r *http.Request) {
	//directory := "update.handlerUpdate"

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "wrong Content-Type", http.StatusBadRequest)
		return
	}

	// http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	prefix := "/update/"
	haspref := strings.HasPrefix(r.URL.Path, prefix)
	if !haspref {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	request := metricRequest{
		typ:   parts[0],
		name:  parts[1],
		value: parts[2],
	}

	if request.name == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	log.Print(parts)
	log.Printf("value bytes: %v, string: %q", []byte(request.value), request.value)
	log.Printf("typ=%q, name=%q, value=%q", request.typ, request.name, request.value)
	switch request.typ {
	case "gauge":
		fGauge, err := strconv.ParseFloat(request.value, 64)
		if err != nil {
			http.Error(w, "status bad request", http.StatusBadRequest)
			//log.Printf()
			return
		}
		err = h.repo.Set(request.name, fGauge)
		if err != nil {
			http.Error(w, "InternalServerError", http.StatusInternalServerError)
			return
		}
	case "counter":
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

func CreateUpdateHandler() UpdateHandler {
	return UpdateHandler{
		repo: db.Create(),
	}
}
