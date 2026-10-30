package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bhuiyanmarzan/olx-api/internal/app"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/config"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/logging"
)

func main() {
	logger := logging.New(os.Stdout)
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	application, err := app.New(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer application.Close()
	server := application.Server()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	logger.Info("server starting", "address", server.Addr, "environment", cfg.Env)
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err = server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return err
	}
	err = <-done
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
