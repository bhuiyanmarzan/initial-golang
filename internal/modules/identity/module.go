package identity

import (
	"net/http"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/httpx"
)

type Module struct{}

func NewModule() *Module { return &Module{} }
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/identity/health", m.status)
	// Preserve the original placeholder endpoint contracts during consolidation.
	mux.HandleFunc("GET /api/v1/auth/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, 200, StatusResponse{Module: "auth", Status: "ok"})
	})
	mux.HandleFunc("GET /api/v1/users/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, 200, StatusResponse{Module: "users", Status: "ok"})
	})
}
func (m *Module) status(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, 200, StatusResponse{Module: "identity", Status: "ok"})
}
