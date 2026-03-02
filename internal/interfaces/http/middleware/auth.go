package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

type tokenVerifier interface {
	Verify(ctx context.Context, token string) (subject string, err error)
}

type AuthMiddleware struct {
	verifier tokenVerifier
	users    user.Repository
}

func NewAuthMiddleware(verifier tokenVerifier, users user.Repository) *AuthMiddleware {
	return &AuthMiddleware{
		verifier: verifier,
		users:    users,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		subject, err := m.verifier.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		requestUser, err := m.users.GetByID(r.Context(), subject)
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), requestUser)))
	})
}

func bearerToken(authorization string) string {
	auth := strings.TrimSpace(authorization)
	if auth == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
}
