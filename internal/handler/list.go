package handler

import (
	"html/template"
	"net/http"

	"go.uber.org/zap"
)

// ListMetrics обрабатывет GET запросы на пути `/`.
// Ничего не принимает.
// Возвращает HTML-страницу со списком всех записанных метрик (Gauge и Counter).
// Возвращает HTTP 500 при неудачном извлечении данных или рендеринга шаблона.
func (ms *MetricService) ListMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   ms.repo.Gauges(r.Context()),
		Counters: ms.repo.Counters(r.Context()),
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		ms.logger.Error("error", zap.Error(err))
		return
	}
}

// tmpl представляет собой HTML-шаблон для отображения страницы метрик.
// Ожидает на вход структуру, содержащую поля Gauges (map[string]float64)
// и Counters (map[string]int64).
var tmpl = template.Must(template.New("metrics").Parse(` 
<html>
<body>
    <h1>Metrics</h1>
    <ul>
    {{range $name, $value := .Gauges}}
        <li>gauge {{$name}} = {{$value}}</li>
    {{end}}
    {{range $name, $value := .Counters}}
        <li>counter {{$name}} = {{$value}}</li>
    {{end}}
    </ul>
</body>
</html>
`))
