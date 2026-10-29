package health

import (
	"net/http"
)

// Module encapsulates the healthcheck domain.
type Module struct{}

// NewModule constructs a new health module.
func NewModule() *Module {
	return &Module{}
}

// RegisterRoutes registers the health routes with the HTTP multiplexer.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", Handler)
}
