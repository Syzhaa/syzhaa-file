package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxJSONBodySize = 1 << 20 // 1MB

// WriteJSON sends a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError sends a JSON error response: {"success":false,"error":msg}.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]interface{}{"success": false, "error": msg})
}

// ReadJSON decodes the request body into v.
// It limits the body to 1MB and rejects unknown fields.
func ReadJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodySize))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}
