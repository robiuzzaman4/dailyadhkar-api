package middleware

import (
	"net/http"
	"strings"
)

const appSecretHeader = "X-App-Secret"

// RequireAppSecret validates that every incoming request carries the correct
// secret in the X-App-Secret header. The expected secret is injected at
// startup from the APP_SECRET environment variable.
func RequireAppSecret(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimSpace(r.Header.Get(appSecretHeader))
		if provided == "" || provided != secret {
			http.Error(w, "unauthorized: missing or invalid app secret", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
