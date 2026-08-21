package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/BREJJNEVV/metrics/internal/handler"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

type flags struct {
	Address string `env:"ADDRESS"`
}

func main() {

	fl := setFlags()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer logger.Sync()

	r := chi.NewRouter()
	r.Use(handler.WithLogging(logger))
	r.Use(middleware.RedirectSlashes)

	service := handler.CreateMetricService()
	r.Post("/update/{type:.*}/{name:.*}/{value:.*}", service.Update)
	r.Post("/update", service.UpdateJSON)
	r.Get("/", service.ListMetrics)

	r.Route("/value", func(r chi.Router) {
		r.Route("/{type:.*}", func(r chi.Router) {
			r.Get("/{name:.*}", service.GetValue)
		})
	})
	r.Get("/value", service.GetValueJSON)

	srv := http.Server{
		Handler: r,
		Addr:    fl.Address,
	}
	logger.Info("Server started", zap.String("address", fl.Address))
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
	if fl.Address == "" {
		fl.Address = *address
	}

	return fl
}
