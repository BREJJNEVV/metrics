package handler

import (
	"database/sql"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type HealthHandler struct {
	db     *sql.DB
	logger *zap.Logger
}

func (hh *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()
	if err := hh.db.PingContext(ctx); err != nil {
		hh.logger.Fatal("DB ping error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hh.logger.Info("db ping ok", zap.Duration("duration", time.Since(start)))
	w.WriteHeader(http.StatusOK)
}

func CreateHealthHandler(db *sql.DB, logger *zap.Logger) HealthHandler {
	return HealthHandler{
		db:     db,
		logger: logger,
	}
}
