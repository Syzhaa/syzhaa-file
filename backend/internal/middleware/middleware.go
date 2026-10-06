package middleware

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := []string{
			"https://ambilfile.web.id",
			"http://localhost:3000",
			"http://localhost:4006",
		}
		// Also allow the configured public base URL (for the user's own domain).
		if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
			allowedOrigins = append(allowedOrigins, baseURL)
		}

		// Check if origin is allowed
		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Session-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://accounts.google.com https://apis.google.com https://cdnjs.cloudflare.com https://static.cloudflareinsights.com; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' https://fonts.gstatic.com; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self'; " +
			"frame-src https://accounts.google.com; " +
			"upgrade-insecure-requests;"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}

// StaticCacheMiddleware sets sane Cache-Control headers for static assets so
// browsers always revalidate HTML/JS/CSS (no more stale-UI confusion) while
// images/fonts can be cached long-term.
func StaticCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		switch ext {
		case ".html", ".js", ".css", ".json", "":
			// Revalidate every time; server replies 304 when unchanged (cheap).
			w.Header().Set("Cache-Control", "no-cache")
		case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico",
			".woff", ".woff2", ".ttf", ".eot", ".mp4", ".webm":
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		next.ServeHTTP(w, r)
	})
}

func CleanURLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if filepath.Ext(path) == "" && path != "/" {
			filePath := filepath.Join("../frontend", path)

			htmlPath := filePath + ".html"
			if _, err := os.Stat(htmlPath); err == nil {
				w.Header().Set("Cache-Control", "no-cache")
				http.ServeFile(w, r, htmlPath)
				return
			}

			indexPath := filepath.Join(filePath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				w.Header().Set("Cache-Control", "no-cache")
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// CsrfOriginMiddleware validates Origin/Referer headers on state-changing requests.
// This is a simpler alternative to token-based CSRF: browsers always send Origin
// on POST/PUT/DELETE, and attackers cannot spoof it cross-origin.
// API key requests (Authorization: Bearer) are exempt (machine-to-machine).
func CsrfOriginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check state-changing methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Exempt API key auth (machine-to-machine, not cookie-based)
		if auth := r.Header.Get("Authorization"); auth != "" {
			next.ServeHTTP(w, r)
			return
		}

		// Build allowed origins list
		allowed := map[string]bool{
			"https://ambilfile.web.id": true,
			"http://localhost:3000":    true,
			"http://localhost:4006":    true,
		}
		if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
			allowed[baseURL] = true
			allowed[strings.TrimSuffix(baseURL, "/")] = true
		}

		// Check Origin header first
		if origin := r.Header.Get("Origin"); origin != "" {
			if allowed[origin] {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, `{"error":"Invalid Origin"}`, http.StatusForbidden)
			return
		}

		// Fallback to Referer header
		if referer := r.Header.Get("Referer"); referer != "" {
			for a := range allowed {
				if strings.HasPrefix(referer, a+"/") || referer == a {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"Invalid Referer"}`, http.StatusForbidden)
			return
		}

		// No Origin or Referer: allow (non-browser clients, curl, etc.)
		// Browsers always send Origin on POST/PUT/DELETE, so this is safe.
		next.ServeHTTP(w, r)
	})
}
