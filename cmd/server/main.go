package main

import (
	"log"
	"net/http"

	"github.com/BREJJNEVV/metrics/internal/handler"
	"github.com/go-chi/chi"
)

func main() {
	r := chi.NewRouter()

	//GET http://<АДРЕС_СЕРВЕРА>/value/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
	service := handler.CreateMetricService()
	r.Post("/update/{type:.*}/{name:.*}/{value:.*}", service.Update)
	r.Get("/", service.ListMetrics)

	r.Route("/value", func(r chi.Router) {
		r.Route("/{type:.*}", func(r chi.Router) {
			r.Get("/{name:.*}", service.GetValue)
		})
	})
	srv := http.Server{
		Handler: r,
		Addr:    ":8080",
	}
	log.Println("Server started at :8080")
	log.Fatal(srv.ListenAndServe())
}
