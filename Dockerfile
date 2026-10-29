# ---------- build ----------
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /bin/api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /bin/migrate ./cmd/migrate

    # ---------- runtime ----------
    FROM gcr.io/distroless/static-debian12:nonroot

    WORKDIR /app
    COPY --from=builder /bin/api /app/bin/api
    COPY --from=builder /bin/migrate /app/bin/migrate
    COPY --from=builder /app/migrations /app/migrations

    EXPOSE 8080
    USER nonroot:nonroot
    ENTRYPOINT ["/app/bin/api"]
