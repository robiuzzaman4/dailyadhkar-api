package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/dailyadhkar-api/internal/domain/user"
)

func registerRoutes(mux *http.ServeMux, db *pgxpool.Pool, users user.Repository) error {
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

	// New direct CRUD routes

	mux.Handle("POST /users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name   string `json:"name"`
			Email  string `json:"email"`
			Gender string `json:"gender"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Email == "" || req.Gender == "" {
			http.Error(w, "name, email, and gender are required", http.StatusBadRequest)
			return
		}

		newID := uuid.New().String()

		createdUser, err := users.Create(r.Context(), user.User{
			ID:                 newID,
			Name:               req.Name,
			Email:              req.Email,
			Gender:             user.Gender(req.Gender),
			IsSubscribed:       true,
			TotalEmailReceived: 0,
			Role:               user.RoleUser,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to create user: %v", err), http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusCreated, createdUser)
	}))

	mux.Handle("GET /users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypassing scope check, list all users for now
		allUsers, err := users.ListByScope(r.Context(), "", user.RoleAdmin)
		if err != nil {
			http.Error(w, "failed to load users", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, map[string]any{
			"users": allUsers,
		})
	}))

	mux.Handle("GET /users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetID := strings.TrimSpace(r.PathValue("id"))
		if targetID == "" {
			http.Error(w, "invalid user id", http.StatusBadRequest)
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
	}))

	mux.Handle("PATCH /users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetID := strings.TrimSpace(r.PathValue("id"))
		if targetID == "" {
			http.Error(w, "invalid user id", http.StatusBadRequest)
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
	}))

	mux.Handle("DELETE /users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetID := strings.TrimSpace(r.PathValue("id"))
		if targetID == "" {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}

		err := users.Delete(r.Context(), targetID)
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to delete user", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))

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
