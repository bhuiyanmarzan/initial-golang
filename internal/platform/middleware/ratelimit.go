package middleware

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/httpx"
)

type clientBucket struct {
	tokens   float64
	lastSeen time.Time
}

// IPRateLimiter provides global in-memory token-bucket rate limiting per IP address.
type IPRateLimiter struct {
	mu             sync.Mutex
	clients        map[string]*clientBucket
	rate           float64 // tokens replenished per second
	capacity       float64 // maximum token capacity (burst limit)
	trustedProxies []netip.Prefix
	skipPaths      map[string]bool
}

type RateLimiterConfig struct {
	RequestsPerMinute int
	Burst             int
	TrustedProxyCIDRs string
	SkipPaths         []string
}

// NewIPRateLimiter creates a new global IP-based rate limiter.
func NewIPRateLimiter(cfg RateLimiterConfig) (*IPRateLimiter, error) {
	rpm := cfg.RequestsPerMinute
	if rpm <= 0 {
		rpm = 120 // Default: 120 requests per minute
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = 30 // Default: burst of 30 requests
	}

	var prefixes []netip.Prefix
	for _, entry := range strings.Split(cfg.TrustedProxyCIDRs, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", entry, err)
		}
		prefixes = append(prefixes, prefix)
	}

	skips := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skips[p] = true
	}
	// Always skip lightweight health probe
	skips["/healthz"] = true

	return &IPRateLimiter{
		clients:        make(map[string]*clientBucket),
		rate:           float64(rpm) / 60.0,
		capacity:       float64(burst),
		trustedProxies: prefixes,
		skipPaths:      skips,
	}, nil
}

// StartCleanup periodically purges inactive clients to prevent memory leaks.
func (l *IPRateLimiter) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				l.mu.Lock()
				for ip, b := range l.clients {
					if now.Sub(b.lastSeen) > 5*time.Minute {
						delete(l.clients, ip)
					}
				}
				l.mu.Unlock()
			}
		}
	}()
}

// Limit returns an HTTP middleware wrapping next with token bucket rate limiting.
func (l *IPRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.skipPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		ip, err := l.resolveClientIP(r)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid request client address", httpx.CodeRateLimited)
			return
		}

		now := time.Now()
		l.mu.Lock()
		bucket, exists := l.clients[ip]
		if !exists {
			bucket = &clientBucket{
				tokens:   l.capacity,
				lastSeen: now,
			}
			l.clients[ip] = bucket
		}

		// Replenish tokens based on elapsed time
		elapsed := now.Sub(bucket.lastSeen).Seconds()
		bucket.tokens = math.Min(l.capacity, bucket.tokens+elapsed*l.rate)
		bucket.lastSeen = now

		if bucket.tokens < 1.0 {
			// Calculate Retry-After in seconds
			secondsToWait := int(math.Ceil((1.0 - bucket.tokens) / l.rate))
			if secondsToWait < 1 {
				secondsToWait = 1
			}
			l.mu.Unlock()

			w.Header().Set("Retry-After", strconv.Itoa(secondsToWait))
			w.Header().Set("Cache-Control", "no-store")
			httpx.Error(w, http.StatusTooManyRequests, "Too many requests; try again later", httpx.CodeRateLimited)
			return
		}

		// Deduct 1 token for the current request
		bucket.tokens -= 1.0
		l.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// resolveClientIP securely extracts the client IP, walking trusted proxy hops from right to left.
func (l *IPRateLimiter) resolveClientIP(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return "", err
	}
	peer = peer.Unmap()

	trusted := func(ip netip.Addr) bool {
		for _, prefix := range l.trustedProxies {
			if prefix.Contains(ip) {
				return true
			}
		}
		return false
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if !trusted(peer) || forwarded == "" {
		return peer.String(), nil
	}

	if len(forwarded) > 2048 {
		return "", fmt.Errorf("forwarded header exceeds length limit")
	}
	hops := strings.Split(forwarded, ",")
	if len(hops) > 32 {
		return "", fmt.Errorf("forwarded header exceeds hop limit")
	}

	for i := len(hops) - 1; i >= 0 && trusted(peer); i-- {
		p, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			return "", fmt.Errorf("invalid hop address %q", hops[i])
		}
		peer = p.Unmap()
	}
	return peer.String(), nil
}
