package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/BREJJNEVV/metrics/internal/handler"
	"github.com/BREJJNEVV/metrics/internal/persistence"
	"github.com/go-chi/chi"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	fl, err := setFlagsEnv()
	if err != nil {
		logger.Fatal("fatal error", zap.Error(err))
	}

	r := chi.NewRouter()
	r.Use(handler.WithLogging(logger))
	r.Use(handler.GzipDecompress)
	r.Use(handler.GzipCompress)

	storage, err := persistence.NewStorage(fl.Restore, fl.StoragePath)
	if err != nil {
		logger.Fatal("fatal error", zap.Error(err))
	}

	var repo handler.Repository = storage

	if fl.Interval > 0 {
		go func() {
			for {
				time.Sleep(time.Duration(fl.Interval * int64(time.Second)))
				err = persistence.SaveMetrics(storage, fl.StoragePath)
				if err != nil {
					logger.Error("save metrics error", zap.Error(err))

				}
			}
		}()
	} else {
		repo = persistence.CreateSyncSaver(storage, fl.StoragePath)
	}
	service := handler.CreateMetricService(repo, logger)

	db, err := sql.Open("pgx", fl.dbDSN)
	if err != nil {
		logger.Fatal("DB init error", zap.Error(err))
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		logger.Warn("db connetion failed", zap.Error(err))
	}

	healthHandler := handler.CreateHealthHandler(db, logger)

	r.Post("/update/{type:.*}/{name:.*}/{value:.*}", service.Update)
	r.Post("/update", service.UpdateJSON)
	r.Post("/update/", service.UpdateJSON)
	r.Get("/", service.ListMetrics)
	r.Get("/ping", healthHandler.Ping)
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
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatal("server stopped with error", zap.Error(err))
	}
}

type flags struct {
	Address     string `env:"ADDRESS"`
	Interval    int64  `env:"STORE_INTERVAL"`
	StoragePath string `env:"FILE_STORAGE_PATH"`
	Restore     bool   `env:"RESTORE"`
	dbDSN       string `env:"DATABASE_DSN"`
}

func setFlagsEnv() (flags, error) {
	address := flag.String("a", "localhost:8080", "endpoint address")
	interval := flag.Int64("i", 300, "store interval")
	storagePath := flag.String("f", "./metrics.json", "storage path")
	restore := flag.Bool("r", false, "restore data or not")
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		`localhost`, `metrics_app`, `AppPassword123`, `metrics`)
	dbDSN := flag.String("d", ps, "address db connection")
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
			return flags{}, fmt.Errorf("invalid STORE_INTERVAL: %w", err)
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
			return flags{}, fmt.Errorf("invalid STORE_INTERVAL: %w", err)
		}
		fl.Restore = v
	} else {
		fl.Restore = *restore
	}
	if env := os.Getenv("DATABASE_DSN"); env != "" {
		fl.dbDSN = env
	} else {
		fl.dbDSN = *dbDSN
	}
	return fl, nil
}
