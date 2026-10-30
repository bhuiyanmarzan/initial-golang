package ratelimit

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"
)

type Policy struct {
	Name       string
	RPM, Burst int
}

func (p Policy) Validate() error {
	if p.Name == "" || p.RPM < 1 || p.Burst < 1 {
		return errors.New("invalid rate policy")
	}
	return nil
}

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}
type Store interface {
	Take(context.Context, string, Policy) (Decision, error)
}

var ErrCapacity = errors.New("limiter capacity reached")

type bucket struct {
	tokens           float64
	updated, expires time.Time
}
type Memory struct {
	mu      sync.Mutex
	buckets map[string]bucket
	max     int
}

func NewMemory(max int) *Memory { return &Memory{buckets: make(map[string]bucket), max: max} }
func (m *Memory) Take(ctx context.Context, key string, p Policy) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	if err := p.Validate(); err != nil {
		return Decision{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	key = p.Name + ":" + key
	b, ok := m.buckets[key]
	if !ok {
		if len(m.buckets) >= m.max {
			return Decision{}, ErrCapacity
		}
		b = bucket{tokens: float64(p.Burst), updated: now}
	}
	rate := float64(p.RPM) / 60
	b.tokens = math.Min(float64(p.Burst), b.tokens+now.Sub(b.updated).Seconds()*rate)
	b.updated = now
	d := Decision{Allowed: b.tokens >= 1}
	if d.Allowed {
		b.tokens--
	} else {
		d.RetryAfter = time.Duration(math.Ceil((1-b.tokens)/rate*1000)) * time.Millisecond
	}
	// Never expire debt before the bucket would be completely replenished.
	b.expires = now.Add(time.Duration(math.Ceil(float64(p.Burst)/rate))*time.Second + time.Minute)
	m.buckets[key] = b
	return d, nil
}
func (m *Memory) Cleanup(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			m.mu.Lock()
			for k, b := range m.buckets {
				if !now.Before(b.expires) {
					delete(m.buckets, k)
				}
			}
			m.mu.Unlock()
		}
	}
}
