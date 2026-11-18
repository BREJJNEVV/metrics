package handler

import (
	"net/http"

	"go.uber.org/zap"
)

type HealthHandler struct {
	pinger Pinger
	logger *zap.Logger
}

// Ping обрабатывает GET запросы на пути /ping.
// Ничего не принимает.
// Возвращает HTTP 200 OK, если сервис работает корректно.
// Возвращает HTTP 500, если проверка состояния завершилась с ошибкой.
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

// NewHealthHandler создает обработчик для проверки /ping.
func NewHealthHandler(p Pinger, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{pinger: p, logger: logger}
}
