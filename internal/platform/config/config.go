package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	Env                  string
	DatabaseURL          string
	MigrationDatabaseURL string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetime    time.Duration
	DBConnMaxIdleTime    time.Duration
	RateLimitRPM         int
	RateLimitBurst       int
	TrustedProxyCIDRs    string
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
}

func MustLoad() Config {
	// Attempt to load .env from current directory, then parent directory
	// _ = godotenv.Load(".env", "../.env")
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		panic("DATABASE_URL is required; use the restricted eghissab_app runtime role")
	}

	migrationDatabaseURL := os.Getenv("MIGRATION_DATABASE_URL")
	if migrationDatabaseURL == "" {
		// Fallback to databaseURL if not separately configured
		migrationDatabaseURL = databaseURL
	}

	return Config{
		Port:                 port,
		Env:                  env,
		DatabaseURL:          databaseURL,
		MigrationDatabaseURL: migrationDatabaseURL,
		DBMaxOpenConns:       getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:       getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime:    getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		DBConnMaxIdleTime:    getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		RateLimitRPM:         getEnvAsInt("RATE_LIMIT_RPM", 120),
		RateLimitBurst:       getEnvAsInt("RATE_LIMIT_BURST", 30),
		TrustedProxyCIDRs:    os.Getenv("AUTH_TRUSTED_PROXY_CIDRS"),
		GoogleClientID:       os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:    os.Getenv("GOOGLE_REDIRECT_URL"),
	}
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
