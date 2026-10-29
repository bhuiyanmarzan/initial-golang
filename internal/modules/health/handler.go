package health

import (
	"net/http"

	"github.com/bhuiyanmarzan/olx-api/internal/platform/httpx"
)

// Handler serves healthcheck probe responses.
func Handler(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
