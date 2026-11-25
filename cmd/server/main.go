package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/BREJJNEVV/metrics/internal/audit"
	"github.com/BREJJNEVV/metrics/internal/handler"
	"github.com/BREJJNEVV/metrics/internal/persistence"
	"github.com/BREJJNEVV/metrics/internal/repository/memory"
	"github.com/BREJJNEVV/metrics/internal/repository/postgres"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"

	_ "net/http/pprof"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var repo handler.Repository
	var pinger handler.Pinger
	var db *pgxpool.Pool
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	fl, err := setFlagsEnv()
	if err != nil {
		logger.Fatal("fatal error", zap.Error(err))
	}
	if fl.dbDSN != "" {
		db, err = pgxpool.New(ctx, fl.dbDSN)
		if err != nil {
			logger.Fatal("failed to create pgx pool", zap.Error(err))
		}
		defer db.Close()

		psgsNew, err := postgres.New(db, logger, fl.dbDSN)
		if err != nil {
			logger.Fatal("init repository error", zap.Error(err))
		}
		repo = psgsNew
		pinger = psgsNew

	} else if fl.StoragePath != "" {
		storage, err := persistence.NewStorage(ctx, fl.Restore, fl.StoragePath)
		if err != nil {
			logger.Fatal("fatal error", zap.Error(err))
		}
		if fl.Interval > 0 {
			prstsNew := storage
			repo = prstsNew
			pinger = prstsNew
			go func() {
				ticker := time.NewTicker(time.Duration(fl.Interval) * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						err := persistence.SaveMetrics(ctx, storage, fl.StoragePath)
						if err != nil {
							logger.Error("save metrics error", zap.Error(err))
						}
					}
				}
			}()
		} else {
			prstsNew := persistence.NewSyncSaver(storage, fl.StoragePath)
			repo = prstsNew
			pinger = prstsNew
		}
	} else {
		mem := memory.Create()
		repo = mem
		pinger = mem
	}

	auditor := audit.NewAuditor(ctx, logger)
	if fl.auditFile != "" {
		fo, err := audit.NewFileObserver(fl.auditFile)
		if err != nil {
			logger.Fatal("observer file error", zap.Error(err))
		}
		auditor.Subscribe(fo)
	}
	if fl.auditURL != "" {
		auditor.Subscribe(audit.NewURLObserver(fl.auditURL, &http.Client{Timeout: 5 * time.Second}))
	}
	service := handler.NewMetricService(repo, logger, auditor)
	healthHandler := handler.NewHealthHandler(pinger, logger)

	r := chi.NewRouter()
	r.Use(handler.WithLogging(logger))
	r.Use(handler.GzipDecompress)
	r.Use(handler.GzipCompress)
	if fl.key != "" {
		r.Use(handler.HashSign(fl.key))
	}

	r.Get("/", service.ListMetrics)
	r.Get("/ping", healthHandler.Ping)
	r.Route("/value", func(r chi.Router) {
		r.Route("/{type:.*}", func(r chi.Router) {
			r.Get("/{name:.*}", service.GetValue)
		})
	})

	r.Group(func(r chi.Router) {
		if fl.key != "" {
			r.Use(handler.HashVerify(fl.key))
		}
		r.Post("/update/{type:.*}/{name:.*}/{value:.*}", service.Update)
		r.Post("/update", service.UpdateJSON)
		r.Post("/update/", service.UpdateJSON)
		r.Post("/updates", service.UpdatesJSON)
		r.Post("/updates/", service.UpdatesJSON)
		r.Post("/value", service.GetValueJSON)
		r.Post("/value/", service.GetValueJSON)
	})

	r.Mount("/debug", middleware.Profiler())
	srv := http.Server{
		Handler: r,
		Addr:    fl.Address,
	}

	logger.Info("Server started", zap.String("address", fl.Address))
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", zap.Error(err))
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(shutdownCtx)
	auditor.Shutdown()
	logger.Info("Shutdown complete")
}

type flags struct {
	Address     string `env:"ADDRESS"`
	StoragePath string `env:"FILE_STORAGE_PATH"`
	dbDSN       string `env:"DATABASE_DSN"`
	key         string `env:"KEY"`
	auditFile   string `env:"AUDIT_FILE"`
	auditURL    string `env:"AUDIT_URL"`
	Interval    int64  `env:"STORE_INTERVAL"`
	Restore     bool   `env:"RESTORE"`
}

func setFlagsEnv() (flags, error) {
	address := flag.String("a", "localhost:8080", "endpoint address")
	interval := flag.Int64("i", 300, "store interval")
	storagePath := flag.String("f", "", "storage path")
	restore := flag.Bool("r", false, "restore data or not")
	dbDSN := flag.String("d", "", "address db connection")
	key := flag.String("k", "", "secret key")
	auditFile := flag.String("audit-file", "", "audit file")
	auditURL := flag.String("audit-url", "", "audit URL")
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
			return flags{}, fmt.Errorf("invalid RESTORE: %w", err)
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

	if env := os.Getenv("KEY"); env != "" {
		fl.key = env
	} else {
		fl.key = *key
	}

	if env := os.Getenv("AUDIT_FILE"); env != "" {
		fl.auditFile = env
	} else {
		fl.auditFile = *auditFile
	}
	if env := os.Getenv("AUDIT_URL"); env != "" {
		fl.auditURL = env
	} else {
		fl.auditURL = *auditURL
	}
	return fl, nil
}
