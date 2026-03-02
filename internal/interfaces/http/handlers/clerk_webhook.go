package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

type webhookVerifier interface {
	Verify(messageID, timestamp, signature string, payload []byte) error
}

type ClerkWebhookHandler struct {
	logger   *slog.Logger
	users    user.Repository
	verifier webhookVerifier
}

func NewClerkWebhookHandler(logger *slog.Logger, users user.Repository, verifier webhookVerifier) *ClerkWebhookHandler {
	return &ClerkWebhookHandler{
		logger:   logger,
		users:    users,
		verifier: verifier,
	}
}

func (h *ClerkWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read webhook body", http.StatusBadRequest)
		return
	}

	if err := h.verifier.Verify(
		r.Header.Get("svix-id"),
		r.Header.Get("svix-timestamp"),
		r.Header.Get("svix-signature"),
		body,
	); err != nil {
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}

	var event clerkWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "user.created", "user.updated":
		if err := h.syncUser(r.Context(), event.Data); err != nil {
			h.logger.Error("failed to sync clerk user", "error", err, "event_type", event.Type, "user_id", event.Data.ID)
			http.Error(w, "failed to sync user", http.StatusInternalServerError)
			return
		}
	default:
		// Ignore unsupported events but acknowledge delivery.
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (h *ClerkWebhookHandler) syncUser(ctx context.Context, data clerkUserData) error {
	email := primaryEmail(data)
	if email == "" {
		return errors.New("missing email in Clerk webhook payload")
	}

	name := displayName(data, email)
	if name == "" {
		return errors.New("missing display name")
	}

	existing, err := h.users.GetByID(ctx, data.ID)
	if errors.Is(err, user.ErrNotFound) {
		_, createErr := h.users.Create(ctx, user.User{
			ID:                 data.ID,
			Name:               name,
			Email:              email,
			IsSubscribed:       true,
			TotalEmailReceived: 0,
			Role:               user.RoleUser,
		})
		return createErr
	}
	if err != nil {
		return err
	}

	existing.Name = name
	existing.Email = email
	_, err = h.users.Update(ctx, *existing)
	return err
}

type clerkWebhookEvent struct {
	Type string        `json:"type"`
	Data clerkUserData `json:"data"`
}

type clerkUserData struct {
	ID                    string              `json:"id"`
	FirstName             string              `json:"first_name"`
	LastName              string              `json:"last_name"`
	Username              string              `json:"username"`
	PrimaryEmailAddressID string              `json:"primary_email_address_id"`
	EmailAddresses        []clerkEmailAddress `json:"email_addresses"`
}

type clerkEmailAddress struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
}

func primaryEmail(data clerkUserData) string {
	if data.PrimaryEmailAddressID != "" {
		for _, email := range data.EmailAddresses {
			if email.ID == data.PrimaryEmailAddressID && strings.TrimSpace(email.EmailAddress) != "" {
				return strings.TrimSpace(email.EmailAddress)
			}
		}
	}

	if len(data.EmailAddresses) == 0 {
		return ""
	}
	return strings.TrimSpace(data.EmailAddresses[0].EmailAddress)
}

func displayName(data clerkUserData, email string) string {
	fullName := strings.TrimSpace(strings.TrimSpace(data.FirstName) + " " + strings.TrimSpace(data.LastName))
	if fullName != "" {
		return fullName
	}

	username := strings.TrimSpace(data.Username)
	if username != "" {
		return username
	}

	at := strings.Index(email, "@")
	if at > 0 {
		return email[:at]
	}

	return strings.TrimSpace(email)
}
