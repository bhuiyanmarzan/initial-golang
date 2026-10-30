package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type ClientIPResolver struct{ trustedProxies []netip.Prefix }

func NewClientIPResolver(cidrs string) (*ClientIPResolver, error) {
	r := &ClientIPResolver{}
	for _, v := range strings.Split(cidrs, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		p, e := netip.ParsePrefix(v)
		if e != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR: %w", e)
		}
		r.trustedProxies = append(r.trustedProxies, p.Masked())
	}
	return r, nil
}

// resolveClientIP securely extracts the client IP, walking trusted proxy hops from right to left.
func (l *ClientIPResolver) Resolve(r *http.Request) (string, error) {
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

	forwarded := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
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
