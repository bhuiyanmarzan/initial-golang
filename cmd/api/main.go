package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/bhuiyanmarzan/olx-api/internal/modules/health"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/config"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/database"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/middleware"
)

func main() {
	cfg := config.MustLoad()

	_, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("main.db.connect: %v", err)
	}

	// Global logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo, // default
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	fmt.Println("database connected")
	fmt.Println("starting olx server...")

	mux := http.NewServeMux()

	healthModule := health.NewModule()
	healthModule.RegisterRoutes(mux)

	// Global middlewares
	rateLimiter, err := middleware.NewIPRateLimiter(middleware.RateLimiterConfig{
		RequestsPerMinute: cfg.RateLimitRPM,
		Burst:             cfg.RateLimitBurst,
		TrustedProxyCIDRs: cfg.TrustedProxyCIDRs,
		SkipPaths:         []string{"/healthz"},
	})
	if err != nil {
		log.Fatalf("rate limiter configuration: %v", err)
	}

	handler := middleware.RequestId(rateLimiter.Limit(mux))

	srv := http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 30,
		IdleTimeout:  time.Second * 60,
	}
	log.Printf("server is listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
