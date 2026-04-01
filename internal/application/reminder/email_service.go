package reminder

import (
	"context"
	"fmt"
	"time"

	"github.com/robiuzzaman4/dailyadhkar-api/internal/domain/user"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/email"
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
	HTML    string
}

type EmailClient interface {
	Send(ctx context.Context, email OutboundEmail) error
}

type EmailService struct {
	client          EmailClient
	sender          string
	maxRetries      int
	retryDelay      time.Duration
	companyName     string
	frontendBaseURL string
}

func NewEmailService(client EmailClient, sender string, companyName string, frontendBaseURL string) *EmailService {
	return &EmailService{
		client:          client,
		sender:          sender,
		maxRetries:      defaultMaxRetries,
		retryDelay:      defaultRetryDelay,
		companyName:     companyName,
		frontendBaseURL: frontendBaseURL,
	}
}

func (s *EmailService) SendDailyAdhkar(ctx context.Context, recipient user.User) error {
	// Render HTML template with user data
	unsubscribeURL := fmt.Sprintf("%s/unsubscribe?email=%s", s.frontendBaseURL, recipient.Email)

	// Determine gender-based greeting
	genderGreeting := "ভাই"
	if string(recipient.Gender) == "female" {
		genderGreeting = "আপু"
	}

	htmlContent, err := email.RenderTemplate(email.TemplateDailyAdhkar, email.TemplateData{
		"name":            recipient.Name,
		"gender":          genderGreeting,
		"company_name":    s.companyName,
		"unsubscribe_url": unsubscribeURL,
	})
	if err != nil {
		return fmt.Errorf("render email template: %w", err)
	}

	// plainTextContent := fmt.Sprintf("Assalamu alaikum %s,\n\nThis is your daily reminder to recite Adhkar.\n\nMay Allah accept your ibadah.", recipient.Name)

	email := OutboundEmail{
		From:    s.sender,
		To:      recipient.Email,
		Subject: "Daily Adhkar Reminder",
		// Text:    plainTextContent,
		HTML: htmlContent,
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
