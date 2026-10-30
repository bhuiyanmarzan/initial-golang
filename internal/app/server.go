package app

import (
	"net/http"
	"time"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/config"
)

func newServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{Addr: ":" + cfg.Port, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
}
