package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/BREJJNEVV/metrics/internal/handler"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi"
)

type flags struct {
	address string `env:"ADDRESS"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fl := setFlags()

	r := chi.NewRouter()

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

	var fl flags
	err := env.Parse(&fl)
	if err != nil {
		log.Fatal(err)
	}
	if fl.address == "" {
		fl.address = *address
	}

	return fl
}
