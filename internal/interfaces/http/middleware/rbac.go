package middleware

import (
	"net/http"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

func RequireRole(allowed ...user.Role) func(http.Handler) http.Handler {
	allowedMap := make(map[user.Role]struct{}, len(allowed))
	for _, role := range allowed {
		allowedMap[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestUser, ok := UserFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if _, ok := allowedMap[requestUser.Role]; !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
