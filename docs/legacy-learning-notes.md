# 1. System Design
# 2. Database Design
# 3. How setup project

### initialize projext
```bash
go mod init github.com/codersgyan/olx-api
```

### create a project entry point
```bash
# main.go
package main

import "fmt"

func main() {
	fmt.Println("Olx-api server is running...")
}
```

### Run this application /first build then run
```bash
go run main.go # build + execute in single command
```

### Production
in production application first build this project then run the application, never run `go run` in production

Build command
```bash
go build -o bin/main main.go # -o Output folder /bin/main
```

Run command
```bash
./bin/main
```

### Makefile (Task Runner Tool)
work with make file
```bash
# Makefile
build:
	go build -o bin/main main.go
run:
	./bin/main
```

Run makefile in bash
```bash
make build  # To build this project
make run    # Run this build file
```
now showing result this type
```text
./bin/main
Olx-api server is running..
```
if you don't want to see this folder name update the Makefile
```bash
# Makefile
build:
	@go build -o bin/main main.go
run:
	@./bin/main
```

if folder name and command name same this showing this error `make: 'build' is up to date. ` to solve this problem 
```makefile
# Makefile
.PHONY: build run
build:
	@go build -o bin/main main.go
run: build # dependency build + run
	@./bin/main
```

fix folder structure every main file create a single binary
```bash
    api     # api separate binary --> bin
    ws      # ws separate binary --> bin
    worker  # worker separate binary --> bin
    cli     # cli separate binary --> bin
/olx-api
    /cmd
        /api
            main.go # This main file create new api binary
```
update makefile
```makefile
.PHONY: build run
build:
	@go build -o bin/api/main ./cmd/api
run: build
	@./bin/api/main
```
initilize git
```bash
git init
```

add .gitignore file
```bash
.gitignore
```
### Build Production-Ready Server using Golang
```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatalf("server failed %v", err)
	}
	fmt.Println("Olx-api server is running...")
}
```
it if bad practise for go http is global Default servemux call, mux = multiplexer

problem
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	http.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatalf("server failed %v", err)
	}
	fmt.Println("Olx-api server is running...")
}
```
then 
```
http://localhost:8090/debug/pprof
```
solution
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	err := http.ListenAndServe(":8090", mux)
	if err != nil {
		log.Fatalf("server failed %v", err)
	}
	fmt.Println("Olx-api server is running...")
}
```
Your remote has commits your local branch doesn't have (often a README or .gitignore added via GitHub's web UI). You need to integrate them before pushing.

Recommended: pull with rebase (keeps history linear)
```bash
git pull --rebase origin main
git push origin main
```
### How Production-Grade Apps Handle Config
```bash
PORT=8080 make run
```
to get this PORT variable
```go
srv := http.Server{
		Addr:         ":" + os.Getenv("PORT"),
		Handler:      mux,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 30,
		IdleTimeout:  time.Second * 60,
	}
```
now problem is lot of env that con't send to the command
```bash
PORT=8080 ............make run
```
solotion .env
```bash
#.env
PORT=8080
```
now start the server
```text
starting olx server...
2026/10/17 08:36:24 server is listening on :
```
port not bind because os.Getenv that can't read the .env file, that possible using file system, for this best practise a package
```text
https://github.com/joho/godotenv
```
indirect packages
```bash
go mod tidy
```

### Production build command
```text
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/api ./cmd/api
```
### Refactore Health function
### Setting up PostgreSQL
### Manage Database Migration for Production
```text
https://github.com/golang-migrate/migrate
```
### Implement Database Migration Rollback Guard
```go
setup
// internal/handlers/listings.go
package handlers

import (
	"database/sql"
	"net/http"
)

func List(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all ok"))
	}
}
```
```go
usages
// cmd/api/main.go
mux.HandleFunc("GET /listings", handlers.List(db))
```
### Fetching Database Records & Returning JSON
### Build DELETE Endpoint with the Constructor Pattern
### Prevent Zombie Queries using Context
Hold the connection 20s this query
```go
	rows, err := lh.db.Query(
		`SELECT id, title, description, price, city, created_at, pg_sleep(20)
			FROM listings
			ORDER BY created_at DESC
			LIMIT 100`)
```
To Check query running or not
```SQL
// seeds/status.sql
SELECT pid, state, query_start, query
FROM pg_stat_activity
WHERE
    query LIKE '%pg_sleep%'
    AND state != 'idle';
```
### Structured logging
by default, this is only strings
```go
// internal/listings.go
	slog.Error("delete failed", "listing_id", id, "err", err)
```
```go
// internal/listings.go
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	log.Error("delete failed", "listing_id", id, "err", err)
```
another way, set default logger
```go
// main.go
// Global logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
```
then use any file
```go
// internal/listings.go
	slog.Error("delete failed", "listing_id", id, "err", err)
```
slog lavel
```go
slog.Debug("debug log", "listing_id", id)
slog.Info("info log", "listing_id", id)
slog.Warn("warn log", "listing_id", id)
slog.Error("error log", "listing_id", id)
```
### Error Envelopes: Standardizing API Error Response
mas assignment problem, here user in input random id, this is the mas assignment problem
```go
type listing struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int64     `json:"price"`
	City        string    `json:"city"`
	CreatedAt   time.Time `json:"created_at"`
}

	var req listing
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lh.logger.Error("failed to decode", "request_id", requestId, "err", err)
		httpx.Error(w, http.StatusBadRequest, "invalid body", httpx.CodeMalformedJson)
		return
	}
```
### Request validation
most popular validation library `go-playground/validator`

### Blackbox testing / API testing
1. Write tests
2. Execute tests
3. Results
