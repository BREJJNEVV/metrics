package handler

import (
	"fmt"
	"net/http"

	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/go-chi/chi"
)

// GetValueJSON обрабатывает POST-запросы на пути /value.
// Принимает тип и имя метрики в формате JSON в теле запроса.
// Требует заголовок Content-Type: application/json.
// Возвращает HTTP 200 и JSON с текущим значением метрики.
// Возвращает HTTP 400 при невалидном запросе или неизвестном типе.
// Возвращает HTTP 404, если метрика с таким именем не найдена.
func (ms *MetricService) GetValue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	typ := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	switch typ {
	case model.Gauge:
		val, found := ms.repo.GetGauge(r.Context(), name)
		if !found {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%g", val)

	case model.Counter:
		val, found := ms.repo.GetCounter(r.Context(), name)
		if !found {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%d", val)

	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
}
