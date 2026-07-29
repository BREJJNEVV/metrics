package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/BREJJNEVV/metrics/internal/handler"
	"github.com/go-chi/chi"
)

type flags struct {
	address string
}

func main() {

	fl := setFlags()

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
		Addr:    fl.address,
	}
	log.Printf("Server started at %s", fl.address)
	log.Fatal(srv.ListenAndServe())
}

func setFlags() flags {
	address := flag.String("a", "localhost:8080", "endpoint address")
	flag.Parse()
	fl := flags{
		address: *address,
	}
	return fl
}
