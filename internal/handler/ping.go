package handler

import (
	"net/http"

	"go.uber.org/zap"
)

type HealthHandler struct {
	pinger Pinger
	logger *zap.Logger
}

func (hh *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := hh.pinger.Ping(ctx)
	if err != nil {
		hh.logger.Error("Ping error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func NewHealthHandler(p Pinger, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{pinger: p, logger: logger}
}
