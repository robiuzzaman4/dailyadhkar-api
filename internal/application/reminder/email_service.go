package reminder

import (
	"context"
	"fmt"
	"time"

	"github.com/robiuzzaman4/dailyadhkar-api/internal/domain/user"
)

const (
	defaultMaxRetries = 3
	defaultRetryDelay = 2 * time.Second
)

type OutboundEmail struct {
	From    string
	To      string
	Subject string
	Text    string
}

type EmailClient interface {
	Send(ctx context.Context, email OutboundEmail) error
}

type EmailService struct {
	client     EmailClient
	sender     string
	maxRetries int
	retryDelay time.Duration
}

func NewEmailService(client EmailClient, sender string) *EmailService {
	return &EmailService{
		client:     client,
		sender:     sender,
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}
}

func (s *EmailService) SendDailyAdhkar(ctx context.Context, recipient user.User) error {
	email := OutboundEmail{
		From:    s.sender,
		To:      recipient.Email,
		Subject: "Daily Adhkar",
		Text:    fmt.Sprintf("Assalamu alaikum %s,\n\nThis is your daily reminder to recite Adhkar.\n\nMay Allah accept your ibadah.", recipient.Name),
	}

	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		if err := s.client.Send(ctx, email); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == s.maxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.retryDelay):
		}
	}

	return fmt.Errorf("send email to %s after %d attempts: %w", recipient.Email, s.maxRetries, lastErr)
}
