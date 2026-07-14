package middleware

import (
	"net/http"
	"os"
	"strings"
)

// devOrigin is the default allowed origin when CORS_ALLOWED_ORIGINS is unset,
// matching Vite's default dev server port.
const devOrigin = "http://localhost:5173"

// CORS wraps next, adding Access-Control-Allow-* headers for origins listed
// in CORS_ALLOWED_ORIGINS (comma-separated) and short-circuiting preflight
// OPTIONS requests before they reach next.
func CORS(next http.Handler) http.Handler {
	allowed := allowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowed(origin, allowed) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		return []string{devOrigin}
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func isAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}
