package middleware

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/httpx"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/ratelimit"
)

type KeyFunc func(*http.Request) (string, error)
type Limiter struct {
	Store       ratelimit.Store
	Fallback    *ratelimit.Memory
	Logger      *slog.Logger
	lastWarning atomic.Int64
}

func (l *Limiter) Wrap(p ratelimit.Policy, key KeyFunc, sensitive bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k, e := key(r)
		if e != nil {
			httpx.Error(w, 400, "invalid request identity", httpx.CodeInvalidID)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 750*time.Millisecond)
		d, e := l.Store.Take(ctx, k, p)
		cancel()
		if e != nil {
			now := time.Now().Unix()
			previous := l.lastWarning.Load()
			if now-previous >= 30 && l.lastWarning.CompareAndSwap(previous, now) {
				l.Logger.Warn("rate limiter unavailable", "policy", p.Name, "fallback", !sensitive)
			}
			if !sensitive && l.Fallback != nil {
				d, e = l.Fallback.Take(r.Context(), k, p)
			}
			if e != nil {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Cache-Control", "no-store")
				httpx.Error(w, 503, "request protection temporarily unavailable", httpx.CodeUnavailable)
				return
			}
		}
		if !d.Allowed {
			seconds := int(math.Ceil(d.RetryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			w.Header().Set("Cache-Control", "no-store")
			httpx.Error(w, 429, "too many requests; try again later", httpx.CodeRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}
