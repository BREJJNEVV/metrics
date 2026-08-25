package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/BREJJNEVV/metrics/internal/handler"
	"github.com/BREJJNEVV/metrics/internal/persistence"
	"github.com/BREJJNEVV/metrics/internal/repository/memory"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func main() {

	fl := setFlagsEnv()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	r := chi.NewRouter()
	r.Use(handler.WithLogging(logger))
	r.Use(handler.GzipDecompress)
	r.Use(handler.GzipCompress)

	storage := memory.Create()
	if fl.Restore {
		err = persistence.LoadMetrics(storage, fl.StoragePath)
		if err != nil {
			log.Fatal(err)
		}
	}
	var repo handler.Repository = storage

	if fl.Interval > 0 {
		go func() {
			for {
				time.Sleep(time.Duration(fl.Interval * int64(time.Second)))
				err = persistence.SaveMetrics(storage, fl.StoragePath)
				if err != nil {
					log.Printf("save metrics error: %v", err)
				}
			}
		}()
	} else {
		repo = persistence.CreateSyncSaver(storage, fl.StoragePath)
	}
	service := handler.CreateMetricService(repo)

	r.Post("/update/{type:.*}/{name:.*}/{value:.*}", service.Update)
	r.Post("/update", service.UpdateJSON)
	r.Post("/update/", service.UpdateJSON)
	r.Get("/", service.ListMetrics)

	r.Route("/value", func(r chi.Router) {
		r.Route("/{type:.*}", func(r chi.Router) {
			r.Get("/{name:.*}", service.GetValue)
		})
	})
	r.Post("/value", service.GetValueJSON)
	r.Post("/value/", service.GetValueJSON)

	srv := http.Server{
		Handler: r,
		Addr:    fl.Address,
	}

	logger.Info("Server started", zap.String("address", fl.Address))
	log.Fatal(srv.ListenAndServe())
}

type flags struct {
	Address     string `env:"ADDRESS"`
	Interval    int64  `env:"STORE_INTERVAL"`
	StoragePath string `env:"FILE_STORAGE_PATH"`
	Restore     bool   `env:"RESTORE"`
}

func setFlagsEnv() flags {
	address := flag.String("a", "localhost:8080", "endpoint address")
	interval := flag.Int64("i", 300, "store interval")
	storagePath := flag.String("f", "./metrics.json", "storage path")
	restore := flag.Bool("r", false, "restore data or not")
	flag.Parse()

	var fl flags
	if env := os.Getenv("ADDRESS"); env != "" {
		fl.Address = env
	} else {
		fl.Address = *address
	}
	if env := os.Getenv("STORE_INTERVAL"); env != "" {
		v, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			log.Fatalf("invalid STORE_INTERVAL: %v", err)
		}
		fl.Interval = v
	} else {
		fl.Interval = *interval
	}
	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		fl.StoragePath = env
	} else {
		fl.StoragePath = *storagePath
	}
	if env := os.Getenv("RESTORE"); env != "" {
		v, err := strconv.ParseBool(env)
		if err != nil {
			log.Fatalf("invalid RESTORE: %v", err)
		}
		fl.Restore = v
	} else {
		fl.Restore = *restore
	}
	return fl
}
