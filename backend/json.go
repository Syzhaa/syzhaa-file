package main

import (
	"encoding/json"
	"net/http"
)

// writeJSON sends a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// okJSON sends a 200 JSON response.
func okJSON(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, data)
}

// errJSON sends a JSON error response.
func errJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{"success": false, "error": msg})
}

// readJSON decodes the request body into v.
func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
