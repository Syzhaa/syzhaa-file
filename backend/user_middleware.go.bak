package main

import (
	"context"
	"net/http"
)

// Middleware: Require user session (approved users only)
func requireUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_session")
		if err != nil {
			http.Redirect(w, r, "/user-login.html", http.StatusTemporaryRedirect)
			return
		}

		user, err := validateUserSession(cookie.Value)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "user_session",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/user-login.html", http.StatusTemporaryRedirect)
			return
		}

		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware: Optional user session (doesn't block, just adds user to context if exists)
func optionalUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_session")
		if err == nil {
			user, err := validateUserSession(cookie.Value)
			if err == nil {
				ctx := context.WithValue(r.Context(), "user", user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
