package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/httpx"
)

type responseWriter struct {
	http.ResponseWriter
	status, bytes int
}

func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *responseWriter) WriteHeader(status int) {
	if status >= 100 && status < 200 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	n, e := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, e
}
func (w *responseWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijacking unavailable")
	}
	return h.Hijack()
}
func Observe(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("request panic", "request_id", RequestIDFromContext(r.Context()))
				if rw.status == 0 {
					httpx.Error(rw, 500, "internal server error", httpx.CodeInternalError)
				} else {
					panic(http.ErrAbortHandler)
				}
			}
			status := rw.status
			if status == 0 {
				status = 200
			}
			logger.Info("http request", "request_id", RequestIDFromContext(r.Context()), "method", r.Method, "route", r.Pattern, "status", status, "bytes", rw.bytes, "duration_ms", time.Since(start).Milliseconds())
		}()
		next.ServeHTTP(rw, r)
	})
}
func BodyLimit(max int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > max {
			httpx.Error(w, 413, "request body too large", httpx.CodeValidationFailed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, max)
		next.ServeHTTP(w, r)
	})
}
