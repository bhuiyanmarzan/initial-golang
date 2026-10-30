package health

import (
	"net/http"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/httpx"
)

type Handler struct{ service *Service }

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, 200, Response{Status: "okay"})
}
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	dto, err := ParseReadinessRequest(r.URL.Query())
	if err != nil {
		httpx.ValidationError(w, 400, err.Error(), httpx.CodeValidationFailed, "dependency")
		return
	}
	result := h.service.Check(r.Context(), dto.toInput())
	w.Header().Set("Cache-Control", "no-store")
	if !result.Ready {
		httpx.Error(w, http.StatusServiceUnavailable, "dependencies not ready", httpx.CodeUnavailable)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(result))
}
