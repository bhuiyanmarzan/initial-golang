.PHONY: build run dev migrate-up migrate-down tidy clean fmt vet

dev:
	air
build:
	@go build -trimpath -o bin/api/main ./cmd/api
	@go build -trimpath -o bin/migrate/main ./cmd/migrate
run: build
	@./bin/api/main
migrate-up:
	@go run ./cmd/migrate up
migrate-down:
	@go run ./cmd/migrate down
fmt:
	gofmt -w cmd internal
vet:
	go vet ./...
tidy:
	go mod tidy
clean:
	rm -rf tmp bin
