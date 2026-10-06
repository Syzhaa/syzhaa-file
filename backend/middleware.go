package main

import (
	"github.com/syzhaa/file-server/internal/auth"
	"context"
	"net/http"
	"strings"
)

// Middleware: Require admin session
func requireAdminSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin, err := auth.ValidateAdminSession(r)
		if err != nil {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "admin", admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware: Require API Key
func requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"Invalid Authorization format. Use: Bearer <api_key>"}`, http.StatusUnauthorized)
			return
		}

		apiKey, err := validateAPIKey(parts[1])
		if err != nil {
			http.Error(w, `{"error":"Invalid or expired API key"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "api_key", apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
