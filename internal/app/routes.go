package app

import (
	"net/http"
	"strings"

	"github.com/bhuiyanmarzan/olx-api/internal/modules/health"
	"github.com/bhuiyanmarzan/olx-api/internal/modules/identity"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/authcontext"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/httpx"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/middleware"
	"github.com/bhuiyanmarzan/olx-api/internal/platform/ratelimit"
)

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	var redisProbe health.Probe
	if a.Redis != nil {
		redisProbe = a.Redis.Ping
	}
	health.NewModule(a.DB.PingContext, redisProbe).RegisterRoutes(mux)
	identity.NewModule().RegisterRoutes(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, 404, "route not found", httpx.CodeNotFound)
	})
	ip := func(r *http.Request) (string, error) { return a.resolver.Resolve(r) }
	public := a.limiter.Wrap(ratelimit.Policy{Name: "public-ip", RPM: a.Config.RateLimitRPM, Burst: a.Config.RateLimitBurst}, ip, false, mux)
	sensitive := a.limiter.Wrap(ratelimit.Policy{Name: "auth-ip", RPM: a.Config.AuthRateLimitRPM, Burst: a.Config.AuthRateLimitBurst}, ip, true, mux)
	dispatch := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only exact GET/HEAD operational probes bypass protection.
		if (r.Method == "GET" || r.Method == "HEAD") && (r.URL.Path == "/healthz" || r.URL.Path == "/api/healthz" || r.URL.Path == "/readyz") {
			mux.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") || strings.HasPrefix(r.URL.Path, "/api/v1/identity/") {
			sensitive.ServeHTTP(w, r)
			return
		}
		public.ServeHTTP(w, r)
	})
	return middleware.RequestId(middleware.Observe(a.Logger, middleware.BodyLimit(a.Config.MaxBodyBytes, dispatch)))
}

// Protected must be placed INSIDE an authentication middleware. Missing verified
// context is rejected; never use raw user/store headers as rate-limit keys.
func (a *App) Protected(next http.Handler) http.Handler {
	limited := a.limiter.Wrap(ratelimit.Policy{Name: "user", RPM: a.Config.UserRateLimitRPM, Burst: a.Config.UserRateLimitBurst}, authcontext.UserKey, false, next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := authcontext.PrincipalFrom(r.Context()); !ok {
			httpx.Error(w, 401, "authentication required", httpx.CodeUnauthenticated)
			return
		}
		limited.ServeHTTP(w, r)
	})
}

// Operation adds a named policy inside authentication for expensive operations.
func (a *App) Operation(p ratelimit.Policy, next http.Handler) (http.Handler, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return a.Protected(a.limiter.Wrap(p, authcontext.UserKey, true, next)), nil
}
