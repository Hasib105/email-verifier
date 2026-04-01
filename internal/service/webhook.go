package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"email-verifier-api/internal/store"
)

type WebhookDispatcher interface {
	Send(ctx context.Context, event string, rec *store.VerificationRecord) error
	SendWithURL(ctx context.Context, event string, rec *store.VerificationRecord, webhookURL string) error
}

type HTTPWebhookDispatcher struct {
	defaultURL string
	client     *http.Client
}

func NewHTTPWebhookDispatcher(url string, timeout time.Duration) *HTTPWebhookDispatcher {
	return &HTTPWebhookDispatcher{
		defaultURL: url,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (d *HTTPWebhookDispatcher) Send(ctx context.Context, event string, rec *store.VerificationRecord) error {
	return d.SendWithURL(ctx, event, rec, "")
}

func (d *HTTPWebhookDispatcher) SendWithURL(ctx context.Context, event string, rec *store.VerificationRecord, webhookURL string) error {
	url := webhookURL
	if url == "" {
		url = d.defaultURL
	}
	if url == "" || rec == nil {
		return nil
	}

	payload := map[string]interface{}{
		"event":             event,
		"id":                rec.ID,
		"email":             rec.Email,
		"status":            rec.Status,
		"message":           rec.Message,
		"source":            rec.Source,
		"user_id":           rec.UserID,
		"check_count":       rec.CheckCount,
		"finalized":         rec.Finalized,
		"checked_at":        rec.LastCheckedAt,
		"confidence":        rec.Confidence,
		"deterministic":     rec.Deterministic,
		"reason_code":       rec.ReasonCode,
		"verification_path": rec.VerificationPath,
		"signal_summary":    rec.SignalSummary,
		"expires_at":        rec.ExpiresAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
