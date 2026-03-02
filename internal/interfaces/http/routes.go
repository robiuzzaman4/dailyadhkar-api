package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
	"github.com/robiuzzaman4/daily-durood-api/internal/infrastructure/auth/clerk"
	"github.com/robiuzzaman4/daily-durood-api/internal/infrastructure/config"
	"github.com/robiuzzaman4/daily-durood-api/internal/interfaces/http/handlers"
	"github.com/robiuzzaman4/daily-durood-api/internal/interfaces/http/middleware"
)

func registerRoutes(mux *http.ServeMux, cfg *config.Config, logger *slog.Logger, db *pgxpool.Pool, users user.Repository) error {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	webhookVerifier, err := clerk.NewWebhookVerifier(cfg.ClerkWebhookSecret)
	if err != nil {
		return fmt.Errorf("create webhook verifier: %w", err)
	}
	mux.Handle("POST /internal/webhooks/clerk", handlers.NewClerkWebhookHandler(logger, users, webhookVerifier))

	tokenVerifier := clerk.NewTokenVerifier(cfg.ClerkJWKSURL, cfg.ClerkIssuer)
	authMW := middleware.NewAuthMiddleware(clerkTokenVerifierAdapter{verifier: tokenVerifier}, users)

	mux.Handle("GET /internal/auth/check", authMW.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		_ = writeJSON(w, http.StatusOK, map[string]any{
			"id":    requestUser.ID,
			"email": requestUser.Email,
			"role":  requestUser.Role,
		})
	})))

	mux.Handle("GET /users/me", authMW.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		profile, err := users.GetByID(r.Context(), requestUser.ID)
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load profile", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, profile)
	})))

	mux.Handle("GET /users", authMW.RequireAuth(middleware.RequireRole(user.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		allUsers, err := users.ListByScope(r.Context(), requestUser.ID, requestUser.Role)
		if err != nil {
			http.Error(w, "failed to load users", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, map[string]any{
			"users": allUsers,
		})
	}))))

	mux.Handle("GET /users/{id}", authMW.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		targetID := strings.TrimSpace(r.PathValue("id"))
		if targetID == "" {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		if requestUser.Role != user.RoleAdmin && requestUser.ID != targetID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		profile, err := users.GetByID(r.Context(), targetID)
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, profile)
	})))

	mux.Handle("PATCH /users/{id}", authMW.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		targetID := strings.TrimSpace(r.PathValue("id"))
		if targetID == "" {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		if requestUser.Role != user.RoleAdmin && requestUser.ID != targetID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var req struct {
			IsSubscribed *bool `json:"is_subscribed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
		if req.IsSubscribed == nil {
			http.Error(w, "is_subscribed is required", http.StatusBadRequest)
			return
		}

		existing, err := users.GetByID(r.Context(), targetID)
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}

		existing.IsSubscribed = *req.IsSubscribed
		updated, err := users.Update(r.Context(), *existing)
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to update user", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, updated)
	})))

	mux.Handle("GET /metadata", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalUsers, err := users.CountUsers(r.Context())
		if err != nil {
			http.Error(w, "failed to load metadata", http.StatusInternalServerError)
			return
		}

		totalEmailsSent, err := users.CountTotalEmailsSent(r.Context())
		if err != nil {
			http.Error(w, "failed to load metadata", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, map[string]int64{
			"total_users":       totalUsers,
			"total_emails_sent": totalEmailsSent,
		})
	}))

	return nil
}
