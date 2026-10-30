package health

import "net/http"

type Module struct{ handler *Handler }

func NewModule(database, redis Probe) *Module {
	return &Module{handler: &Handler{service: NewService(database, redis)}}
}
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", m.handler.Live)
	mux.HandleFunc("GET /api/healthz", m.handler.Live)
	mux.HandleFunc("GET /readyz", m.handler.Ready)
}
