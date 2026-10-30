package health

import (
	"context"
	"errors"
	"time"
)

type Probe func(context.Context) error
type CheckInput struct{ Dependency string }
type Result struct{ Ready bool }
type Service struct{ database, redis Probe }

func NewService(database, redis Probe) *Service { return &Service{database: database, redis: redis} }
func (s *Service) Check(ctx context.Context, in CheckInput) Result {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	check := func(p Probe) error {
		if p == nil {
			return errors.New("dependency not configured")
		}
		return p(ctx)
	}
	switch in.Dependency {
	case "database":
		return Result{Ready: check(s.database) == nil}
	case "redis":
		return Result{Ready: check(s.redis) == nil}
	case "":
		if check(s.database) != nil {
			return Result{}
		}
		if s.redis != nil && check(s.redis) != nil {
			return Result{}
		}
		return Result{Ready: true}
	default:
		return Result{}
	}
}
