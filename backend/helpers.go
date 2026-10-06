package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxJSONBodySize = 1 << 20 // 1MB

// writeJSON sends a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError sends a JSON error response: {"success":false,"error":msg}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{"success": false, "error": msg})
}

// readJSON decodes the request body into v.
// It limits the body to 1MB and rejects unknown fields.
func readJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodySize))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}
