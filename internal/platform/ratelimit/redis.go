package ratelimit

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Redis implements only the RESP2 operations needed by the limiter. Connections
// are short-lived, bounded, and deadline-controlled; no automatic request retries.
type Redis struct {
	address, username, password, prefix string
	db                                  int
	tlsConfig                           *tls.Config
	slots                               chan struct{}
}

func NewRedis(raw, prefix string) (*Redis, error) {
	u, e := url.Parse(raw)
	if e != nil {
		return nil, fmt.Errorf("invalid Redis URL")
	}
	if (u.Scheme != "redis" && u.Scheme != "rediss") || u.Hostname() == "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid Redis URL")
	}
	port := u.Port()
	if port == "" {
		port = "6379"
	}
	r := &Redis{address: net.JoinHostPort(u.Hostname(), port), prefix: prefix, slots: make(chan struct{}, 32)}
	if u.User != nil {
		r.username = u.User.Username()
		r.password, _ = u.User.Password()
	}
	if u.Path != "" && u.Path != "/" {
		r.db, e = strconv.Atoi(strings.TrimPrefix(u.Path, "/"))
		if e != nil || r.db < 0 {
			return nil, fmt.Errorf("invalid Redis database")
		}
	}
	if u.Scheme == "rediss" {
		r.tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: u.Hostname()}
	}
	return r, nil
}
func (r *Redis) command(ctx context.Context, args ...string) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	dialer := &net.Dialer{Timeout: 500 * time.Millisecond}
	var c net.Conn
	var err error
	if r.tlsConfig != nil {
		c, err = (&tls.Dialer{NetDialer: dialer, Config: r.tlsConfig}).DialContext(ctx, "tcp", r.address)
	} else {
		c, err = dialer.DialContext(ctx, "tcp", r.address)
	}
	if err != nil {
		return nil, fmt.Errorf("Redis connection unavailable")
	}
	defer c.Close()
	deadline, _ := ctx.Deadline()
	if err = c.SetDeadline(deadline); err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	reader := bufio.NewReader(io.LimitReader(c, 65536))
	writer := bufio.NewWriter(c)
	send := func(parts ...string) (any, error) {
		if _, e := fmt.Fprintf(writer, "*%d\r\n", len(parts)); e != nil {
			return nil, e
		}
		for _, part := range parts {
			if _, e := fmt.Fprintf(writer, "$%d\r\n%s\r\n", len(part), part); e != nil {
				return nil, e
			}
		}
		if e := writer.Flush(); e != nil {
			return nil, e
		}
		return readRESP(reader, 0)
	}
	if r.password != "" || r.username != "" {
		auth := []string{"AUTH", r.password}
		if r.username != "" {
			auth = []string{"AUTH", r.username, r.password}
		}
		if _, err = send(auth...); err != nil {
			return nil, err
		}
	}
	if r.db != 0 {
		if _, err = send("SELECT", strconv.Itoa(r.db)); err != nil {
			return nil, err
		}
	}
	return send(args...)
}
func readRESP(r *bufio.Reader, depth int) (any, error) {
	if depth > 3 {
		return nil, fmt.Errorf("Redis response nesting exceeded")
	}
	line, e := r.ReadString('\n')
	if e != nil {
		return nil, e
	}
	if len(line) < 3 || !strings.HasSuffix(line, "\r\n") {
		return nil, fmt.Errorf("invalid Redis response")
	}
	s := line[1 : len(line)-2]
	switch line[0] {
	case '+':
		return s, nil
	case '-':
		return nil, fmt.Errorf("Redis command failed")
	case ':':
		return strconv.ParseInt(s, 10, 64)
	case '*':
		n, e := strconv.Atoi(s)
		if e != nil || n < 0 || n > 16 {
			return nil, fmt.Errorf("invalid Redis array")
		}
		a := make([]any, n)
		for i := range a {
			a[i], e = readRESP(r, depth+1)
			if e != nil {
				return nil, e
			}
		}
		return a, nil
	default:
		return nil, fmt.Errorf("unsupported Redis response")
	}
}
func (r *Redis) Ping(ctx context.Context) error {
	v, e := r.command(ctx, "PING")
	if e == nil && v != "PONG" {
		return fmt.Errorf("unexpected Redis ping response")
	}
	return e
}

// Redis TIME gives all application instances the same bucket clock.
const tokenScript = `
local tm=redis.call('TIME')
local now=tonumber(tm[1])*1000+tonumber(tm[2])/1000
local rate=tonumber(ARGV[1])/60000
local cap=tonumber(ARGV[2])
local b=redis.call('HMGET',KEYS[1],'tokens','updated')
local tokens=tonumber(b[1]) or cap
local updated=tonumber(b[2]) or now
tokens=math.min(cap,tokens+math.max(0,now-updated)*rate)
local allowed=0
local retry=0
if tokens>=1 then tokens=tokens-1 allowed=1 else retry=math.ceil((1-tokens)/rate) end
redis.call('HSET',KEYS[1],'tokens',tokens,'updated',now)
redis.call('PEXPIRE',KEYS[1],math.ceil(cap/rate)+60000)
return {allowed,retry}
`

func (r *Redis) Take(ctx context.Context, key string, p Policy) (Decision, error) {
	if e := p.Validate(); e != nil {
		return Decision{}, e
	}
	digest := sha256.Sum256([]byte(key))
	k := fmt.Sprintf("%s:%s:%x", r.prefix, p.Name, digest)
	v, e := r.command(ctx, "EVAL", tokenScript, "1", k, strconv.Itoa(p.RPM), strconv.Itoa(p.Burst))
	if e != nil {
		return Decision{}, e
	}
	a, ok := v.([]any)
	if !ok || len(a) != 2 {
		return Decision{}, fmt.Errorf("invalid Redis decision")
	}
	allowed, ok := a[0].(int64)
	retry, ok2 := a[1].(int64)
	if !ok || !ok2 || (allowed != 0 && allowed != 1) || retry < 0 || retry > 60000 {
		return Decision{}, fmt.Errorf("invalid Redis decision")
	}
	return Decision{Allowed: allowed == 1, RetryAfter: time.Duration(retry) * time.Millisecond}, nil
}
