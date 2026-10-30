package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func run() error {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		return fmt.Errorf("usage: migrate <up | down>")
	}
	_ = godotenv.Load()
	dsn := os.Getenv("MIGRATION_DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("MIGRATION_DATABASE_URL is required; runtime credentials are never used for migrations")
	}
	if os.Args[1] == "down" && os.Getenv("ENV") != "development" && os.Getenv("ALLOW_MIGRATION_DOWN") != "true" {
		return fmt.Errorf("rollback outside development requires ALLOW_MIGRATION_DOWN=true")
	}
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("migration initialization failed; verify database and migration configuration")
	}
	defer m.Close()
	if os.Args[1] == "up" {
		err = m.Up()
	} else {
		err = m.Steps(-1)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
