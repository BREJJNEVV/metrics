package handler

import (
	"html/template"
	"log"
	"net/http"
)

func (ms *MetricService) ListMetrics(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Printf("error: %v", err)
		return
	}
}

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
