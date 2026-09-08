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
	"github.com/BREJJNEVV/metrics/internal/repository/memory"
	"github.com/BREJJNEVV/metrics/internal/repository/postgres"
	"github.com/go-chi/chi"
	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	var repo handler.Repository
	var db *sql.DB

	if fl.dbDSN != "" {
		err = runMigrations(fl.dbDSN)
		if err != nil {
			logger.Fatal("failed to run migrations", zap.Error(err))
		}

		db, err = sql.Open("pgx", fl.dbDSN)
		if err != nil {
			logger.Fatal("fatal error", zap.Error(err))
		}
		defer db.Close()

		repo = postgres.New(db, logger)

	} else if fl.StoragePath != "" {
		storage, err := persistence.NewStorage(fl.Restore, fl.StoragePath)
		if err != nil {
			logger.Fatal("fatal error", zap.Error(err))
		}
		if fl.Interval > 0 {
			repo = storage
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
	} else {
		repo = memory.Create()
	}

	service := handler.CreateMetricService(repo, logger)

	r := chi.NewRouter()
	r.Use(handler.WithLogging(logger))
	r.Use(handler.GzipDecompress)
	r.Use(handler.GzipCompress)

	if db != nil {
		if err := db.Ping(); err != nil {
			logger.Warn("db connetion failed", zap.Error(err))
		}
		healthHandler := handler.CreateHealthHandler(db, logger)
		r.Get("/ping", healthHandler.Ping)
	}
	r.Post("/update/{type:.*}/{name:.*}/{value:.*}", service.Update)
	r.Post("/update", service.UpdateJSON)
	r.Post("/update/", service.UpdateJSON)
	r.Post("/updates", service.UpdatesJSON)
	r.Post("/updates/", service.UpdatesJSON)
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
	storagePath := flag.String("f", "", "storage path")
	restore := flag.Bool("r", false, "restore data or not")
	dbDSN := flag.String("d", "", "address db connection")
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

	env := os.Getenv("DATABASE_DSN")

	if env != "" {
		fl.dbDSN = env
	} else if *dbDSN != "" {
		fl.dbDSN = *dbDSN
	} else {
		fl.dbDSN = ""
	}
	return fl, nil
}

func runMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
