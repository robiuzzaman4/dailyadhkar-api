package middleware

import (
	"context"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

type contextKey string

const userContextKey contextKey = "request_user"

func WithUser(ctx context.Context, u *user.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

func UserFromContext(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(userContextKey).(*user.User)
	if !ok || u == nil {
		return nil, false
	}
	return u, true
}
