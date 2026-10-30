package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port, Env, DatabaseURL, MigrationDatabaseURL                                                                                  string
	DBMaxOpenConns, DBMaxIdleConns                                                                                                int
	DBConnMaxLifetime, DBConnMaxIdleTime                                                                                          time.Duration
	RateLimitRPM, RateLimitBurst, AuthRateLimitRPM, AuthRateLimitBurst, UserRateLimitRPM, UserRateLimitBurst, RateLimitMaxClients int
	RateLimitBackend, RedisURL, RedisPrefix, TrustedProxyCIDRs                                                                    string
	MaxBodyBytes                                                                                                                  int64
	ShutdownTimeout                                                                                                               time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()
	c := Config{Port: value("PORT", "8080"), Env: value("ENV", "development"), DatabaseURL: os.Getenv("DATABASE_URL"), MigrationDatabaseURL: os.Getenv("MIGRATION_DATABASE_URL"), RateLimitBackend: value("RATE_LIMIT_BACKEND", "memory"), RedisURL: os.Getenv("REDIS_URL"), RedisPrefix: value("REDIS_PREFIX", "eghissab:ratelimit"), TrustedProxyCIDRs: os.Getenv("AUTH_TRUSTED_PROXY_CIDRS")}
	var err error
	integer := func(name string, def int) int {
		if err != nil {
			return 0
		}
		v, e := strconv.Atoi(value(name, strconv.Itoa(def)))
		if e != nil || v < 1 || v > 100000000 {
			err = fmt.Errorf("%s must be a positive integer no greater than 100000000", name)
		}
		return v
	}
	duration := func(name string, def time.Duration) time.Duration {
		if err != nil {
			return 0
		}
		v, e := time.ParseDuration(value(name, def.String()))
		if e != nil || v <= 0 {
			err = fmt.Errorf("%s must be a positive duration", name)
		}
		return v
	}
	c.DBMaxOpenConns = integer("DB_MAX_OPEN_CONNS", 25)
	c.DBMaxIdleConns = integer("DB_MAX_IDLE_CONNS", 25)
	c.DBConnMaxLifetime = duration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	c.DBConnMaxIdleTime = duration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)
	c.RateLimitRPM = integer("RATE_LIMIT_RPM", 120)
	c.RateLimitBurst = integer("RATE_LIMIT_BURST", 30)
	c.AuthRateLimitRPM = integer("AUTH_RATE_LIMIT_RPM", 10)
	c.AuthRateLimitBurst = integer("AUTH_RATE_LIMIT_BURST", 5)
	c.UserRateLimitRPM = integer("USER_RATE_LIMIT_RPM", 300)
	c.UserRateLimitBurst = integer("USER_RATE_LIMIT_BURST", 60)
	c.RateLimitMaxClients = integer("RATE_LIMIT_MAX_CLIENTS", 10000)
	c.MaxBodyBytes = int64(integer("HTTP_MAX_BODY_BYTES", 1048576))
	c.ShutdownTimeout = duration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}
	port, e := strconv.Atoi(c.Port)
	if e != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}
	if c.Env != "development" && c.Env != "production" && c.Env != "staging" {
		return Config{}, fmt.Errorf("ENV must be development, staging, or production")
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns {
		return Config{}, fmt.Errorf("DB_MAX_IDLE_CONNS exceeds DB_MAX_OPEN_CONNS")
	}
	if c.RateLimitBackend != "memory" && c.RateLimitBackend != "redis" {
		return Config{}, fmt.Errorf("RATE_LIMIT_BACKEND must be memory or redis")
	}
	if c.Env != "development" && c.RateLimitBackend != "redis" {
		return Config{}, fmt.Errorf("Redis rate limiting is required outside development")
	}
	if c.RateLimitBackend == "redis" && c.RedisURL == "" {
		return Config{}, fmt.Errorf("REDIS_URL is required")
	}
	return c, nil
}
func value(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
