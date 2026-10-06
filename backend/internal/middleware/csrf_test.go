package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCsrfOriginMiddleware(t *testing.T) {
	handler := CsrfOriginMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		method     string
		origin     string
		referer    string
		authHeader string
		wantCode   int
	}{
		{"GET allowed", "GET", "", "", "", 200},
		{"POST valid Origin", "POST", "https://ambilfile.web.id", "", "", 200},
		{"POST invalid Origin", "POST", "https://evil.com", "", "", 403},
		{"POST valid Referer", "POST", "", "https://ambilfile.web.id/room.html", "", 200},
		{"POST invalid Referer", "POST", "", "https://evil.com/", "", 403},
		{"POST no headers (non-browser)", "POST", "", "", "", 200},
		{"POST with API key exempt", "POST", "https://evil.com", "", "Bearer sfa_xxx", 200},
		{"PUT valid Origin", "PUT", "https://ambilfile.web.id", "", "", 200},
		{"DELETE invalid Origin", "DELETE", "https://evil.com", "", "", 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}
