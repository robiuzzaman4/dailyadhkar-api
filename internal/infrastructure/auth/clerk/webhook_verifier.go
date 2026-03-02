package clerk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const defaultWebhookTolerance = 5 * time.Minute

type WebhookVerifier struct {
	secret    []byte
	tolerance time.Duration
	now       func() time.Time
}

func NewWebhookVerifier(secret string) (*WebhookVerifier, error) {
	decoded, err := decodeWebhookSecret(secret)
	if err != nil {
		return nil, err
	}

	return &WebhookVerifier{
		secret:    decoded,
		tolerance: defaultWebhookTolerance,
		now:       time.Now,
	}, nil
}

func (v *WebhookVerifier) Verify(messageID, timestamp, signature string, payload []byte) error {
	if messageID == "" || timestamp == "" || signature == "" {
		return errors.New("missing svix headers")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid svix timestamp")
	}
	signedAt := time.Unix(ts, 0)
	now := v.now()
	if signedAt.Before(now.Add(-v.tolerance)) || signedAt.After(now.Add(v.tolerance)) {
		return errors.New("svix timestamp outside tolerance")
	}

	message := messageID + "." + timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(message))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, provided := range extractV1Signatures(signature) {
		if hmac.Equal([]byte(provided), []byte(expected)) {
			return nil
		}
	}

	return errors.New("invalid svix signature")
}

func decodeWebhookSecret(secret string) ([]byte, error) {
	clean := strings.TrimSpace(secret)
	clean = strings.TrimPrefix(clean, "whsec_")

	decoded, err := base64.StdEncoding.DecodeString(clean)
	if err == nil {
		return decoded, nil
	}

	decoded, rawErr := base64.RawStdEncoding.DecodeString(clean)
	if rawErr != nil {
		return nil, fmt.Errorf("decode webhook secret: %w", err)
	}

	return decoded, nil
}

func extractV1Signatures(header string) []string {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil
	}

	signatures := make([]string, 0)
	for _, part := range strings.Fields(header) {
		seg := strings.SplitN(part, ",", 2)
		if len(seg) != 2 {
			continue
		}
		if strings.TrimSpace(seg[0]) != "v1" {
			continue
		}
		signatures = append(signatures, strings.TrimSpace(seg[1]))
	}

	// Fallback for comma-only formatting: "v1,sig1,v1,sig2"
	if len(signatures) == 0 {
		parts := strings.Split(header, ",")
		for i := 0; i+1 < len(parts); i++ {
			version := strings.TrimSpace(parts[i])
			value := strings.TrimSpace(parts[i+1])
			if version == "v1" && value != "" {
				signatures = append(signatures, value)
			}
		}
	}

	return signatures
}
