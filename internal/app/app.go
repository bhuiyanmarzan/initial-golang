package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/config"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/database"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/middleware"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/ratelimit"
)

type App struct {
	Config       config.Config
	DB           *sql.DB
	Transactions *database.Transactor
	Logger       *slog.Logger
	Redis        *ratelimit.Redis
	limiter      *middleware.Limiter
	resolver     *middleware.ClientIPResolver
	cancel       context.CancelFunc
	cleanupDone  chan struct{}
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	resolver, err := middleware.NewClientIPResolver(cfg.TrustedProxyCIDRs)
	if err != nil {
		return nil, err
	}
	db, err := database.ConnectWithPool(cfg.DatabaseURL, database.PoolConfig{MaxOpenConns: cfg.DBMaxOpenConns, MaxIdleConns: cfg.DBMaxIdleConns, ConnMaxLifetime: cfg.DBConnMaxLifetime, ConnMaxIdleTime: cfg.DBConnMaxIdleTime})
	if err != nil {
		return nil, err
	}
	a := &App{Config: cfg, DB: db, Transactions: database.NewTransactor(db), Logger: logger, resolver: resolver, cleanupDone: make(chan struct{})}
	memory := ratelimit.NewMemory(cfg.RateLimitMaxClients)
	var store ratelimit.Store = memory
	if cfg.RateLimitBackend == "redis" {
		a.Redis, err = ratelimit.NewRedis(cfg.RedisURL, cfg.RedisPrefix)
		if err == nil {
			err = a.Redis.Ping(ctx)
		}
		if err != nil {
			_ = db.Close()
			return nil, err
		}
		store = a.Redis
	}
	a.limiter = &middleware.Limiter{Store: store, Logger: logger}
	if a.Redis != nil {
		a.limiter.Fallback = memory
	}
	cleanupCtx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	go func() { defer close(a.cleanupDone); memory.Cleanup(cleanupCtx) }()
	logger.Info("rate limiter configured", "backend", cfg.RateLimitBackend, "rpm", cfg.RateLimitRPM, "burst", cfg.RateLimitBurst, "max_local_clients", cfg.RateLimitMaxClients)
	return a, nil
}
func (a *App) Close() error         { a.cancel(); <-a.cleanupDone; return a.DB.Close() }
func (a *App) Server() *http.Server { return newServer(a.Config, a.routes()) }
