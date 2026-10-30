package httpx

import (
	"encoding/json"
	"net/http"
)

type Code string

const (
	CodeInvalidID        Code = "invalid_id"
	CodeInternalError    Code = "internal_error"
	CodeNotFound         Code = "not_found"
	CodeMalformedJson    Code = "malformed_json"
	CodeValidationFailed Code = "validation_failed"
	CodeUnauthenticated  Code = "unauthenticated"
	CodeForbidden        Code = "forbidden"
	CodeConflict         Code = "conflict"
	CodeRateLimited      Code = "rate_limited"
	CodeUnavailable      Code = "unavailable"
)

type errorEnvelop struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func Error(w http.ResponseWriter, status int, message string, code Code) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// _ = json.NewEncoder(w).Encode(map[string]string{"message": message, "code": code})
	_ = json.NewEncoder(w).Encode(errorEnvelop{Error: errorPayload{
		Code:    code,
		Message: message,
	}})
}

func ValidationError(w http.ResponseWriter, status int, message string, code Code, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// _ = json.NewEncoder(w).Encode(map[string]string{"message": message, "code": code})
	_ = json.NewEncoder(w).Encode(errorEnvelop{Error: errorPayload{
		Code:    code,
		Message: message,
		Field:   field,
	}})
}
