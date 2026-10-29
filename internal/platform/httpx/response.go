package httpx

import (
	"encoding/json"
	"net/http"
)

// successEnvelop wraps every successful JSON response.
type successEnvelop[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

// JSON writes a successful JSON response wrapped in the global envelope.
// status is the HTTP status code (e.g. http.StatusOK, http.StatusCreated).
func JSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(successEnvelop[T]{Success: true, Data: data})
}

// NoContent writes a 204 No Content response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
