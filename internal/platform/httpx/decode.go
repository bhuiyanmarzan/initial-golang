package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// DecodeJSON rejects unknown fields, extra JSON values and unsupported media types.
// Body size is bounded by the global BodyLimit middleware.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		Error(w, 415, "Content-Type must be application/json", CodeMalformedJson)
		return false
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err = d.Decode(dst)
	if err == nil {
		if e := d.Decode(new(any)); e != io.EOF {
			if e == nil {
				err = errors.New("multiple JSON values")
			} else {
				err = e
			}
		}
	}
	if err != nil {
		var size *http.MaxBytesError
		if errors.As(err, &size) {
			Error(w, 413, "request body too large", CodeValidationFailed)
		} else {
			Error(w, 400, "invalid JSON request body", CodeMalformedJson)
		}
		return false
	}
	return true
}

// Optional distinguishes omitted, explicit null and a supplied PATCH value.
type Optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	o.Null = string(data) == "null"
	var zero T
	o.Value = zero
	if o.Null {
		return nil
	}
	return json.Unmarshal(data, &o.Value)
}
