package health

import (
	"fmt"
	"net/url"
)

// ReadinessRequest demonstrates transport validation without leaking HTTP into services.
// With no dependency specified, all configured dependencies are checked.
type ReadinessRequest struct{ Dependency string }

func ParseReadinessRequest(q url.Values) (ReadinessRequest, error) {
	for k, v := range q {
		if k != "dependency" || len(v) != 1 {
			return ReadinessRequest{}, fmt.Errorf("only one dependency parameter is supported")
		}
	}
	d := ReadinessRequest{Dependency: q.Get("dependency")}
	if d.Dependency != "" && d.Dependency != "database" && d.Dependency != "redis" {
		return d, fmt.Errorf("dependency must be database or redis")
	}
	return d, nil
}

type Response struct {
	Status string `json:"status"`
}
