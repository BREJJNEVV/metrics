package handler

import (
	"net/http"
	"text/template"
)

func (ms *MetricService) ListMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   ms.repo.Gauges(),
		Counters: ms.repo.Counters(),
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}

// metrics - имя шаблона
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
