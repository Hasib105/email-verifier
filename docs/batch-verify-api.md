# Batch Verify API Integration Guide

This guide shows how backend services can call `/verify/batch` safely with the hardened V1 response model and webhook-first async updates.

## Endpoint

- Method: `POST`
- URL: `/verify/batch`
- Header: `X-API-Key: <user-api-key>`
- Content type: `application/json`

## Request

```json
{
  "emails": [
    "alice@example.com",
    "bob@example.com",
    "not-an-email"
  ]
}
```

Rules:

- Minimum `1` email
- Maximum `1000` emails
- Each email is processed independently

## Response

```json
{
  "total": 3,
  "accepted": 3,
  "items": [
    {
      "id": "2f06116d-4f3e-4f76-b671-71888fadb5f4",
      "email": "alice@example.com",
      "status": "valid",
      "message": "250 recipient accepted",
      "source": "direct-smtp-check",
      "cached": false,
      "finalized": true,
      "confidence": "medium",
      "deterministic": false,
      "reason_code": "direct_accept_non_strict",
      "verification_path": "direct_smtp",
      "signal_summary": "Recipient MX accepted RCPT on a non-strict provider.",
      "expires_at": 1775259200
    },
    {
      "id": "f1d4f67e-54d4-437e-a5d1-3f89c53f6ff9",
      "email": "not-an-email",
      "status": "invalid",
      "message": "invalid syntax",
      "source": "direct-smtp-check",
      "cached": false,
      "finalized": true,
      "confidence": "high",
      "deterministic": true,
      "reason_code": "syntax_invalid",
      "verification_path": "direct_smtp",
      "signal_summary": "Address failed syntax validation before any network checks.",
      "expires_at": 1775604800
    }
  ]
}
```

Notes:

- `accepted` means the API processed the item, not that the mailbox is `valid`.
- Use `status` together with `confidence`, `deterministic`, and `signal_summary`.
- `pending_bounce_check` means the probe workflow is still waiting for the first bounce window.
- A probe result can be `valid` with `finalized=false` after the first no-bounce check; keep listening for the second check.

## Error Responses

- `401 Unauthorized`: missing or invalid API key
- `400 Bad Request`: invalid JSON, empty list, or batch size over 1000

Example:

```json
{
  "error": "batch limit exceeded: max 1000 emails per request"
}
```

## Recommended Service Pattern

1. Split source emails into chunks of at most `1000`.
2. Submit each chunk to `/verify/batch`.
3. Persist `id`, `email`, `status`, `confidence`, `reason_code`, and `expires_at`.
4. Use finalized results immediately.
5. Configure `/users/webhook` and process webhook events for items that remain `pending_bounce_check` or have `finalized=false`.
6. Use `GET /verifications/{id}` only for reconciliation if your webhook receiver was temporarily unavailable.
7. Re-query expired items instead of assuming the old result still holds.

## Go Example

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

type BatchRequest struct {
	Emails []string `json:"emails"`
}

type VerifyItem struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	Source           string `json:"source"`
	Cached           bool   `json:"cached"`
	Finalized        bool   `json:"finalized"`
	NextCheckAt      int64  `json:"next_check_at,omitempty"`
	Confidence       string `json:"confidence"`
	Deterministic    bool   `json:"deterministic"`
	ReasonCode       string `json:"reason_code"`
	VerificationPath string `json:"verification_path"`
	SignalSummary    string `json:"signal_summary"`
	ExpiresAt        int64  `json:"expires_at"`
}

type BatchResponse struct {
	Total    int          `json:"total"`
	Accepted int          `json:"accepted"`
	Items    []VerifyItem `json:"items"`
}

type ListVerificationsResponse struct {
	Items []VerifyItem `json:"items"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) doJSON(method, path string, reqBody any, out any) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.APIKey)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s failed: status=%d body=%s", method, path, resp.StatusCode, string(raw))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}

	return nil
}

func (c *Client) VerifyBatch(emails []string) (*BatchResponse, error) {
	var out BatchResponse
	if err := c.doJSON(http.MethodPost, "/verify/batch", BatchRequest{Emails: emails}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetVerificationByID(id string) (*VerifyItem, error) {
	var out VerifyItem
	if err := c.doJSON(http.MethodGet, "/verifications/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListVerifications(limit, offset int) (*ListVerificationsResponse, error) {
	q := url.Values{}
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))

	var out ListVerificationsResponse
	if err := c.doJSON(http.MethodGet, "/verifications?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateWebhook(webhookURL string) error {
	req := map[string]string{"webhook_url": webhookURL}
	return c.doJSON(http.MethodPut, "/users/webhook", req, nil)
}

func (c *Client) SendTestWebhook(webhookURL string) error {
	req := map[string]string{"webhook_url": webhookURL}
	return c.doJSON(http.MethodPost, "/users/webhook/test", req, nil)
}
```

## Webhook-First Result Flow

1. Call `/verify/batch` and store `items[i].id`.
2. Use items with `finalized=true` immediately.
3. For items with `status=pending_bounce_check` or `finalized=false`, wait for webhook events and update your stored result by `id`.
4. If webhook delivery is interrupted, reconcile via `GET /verifications/{id}`.
5. Re-verify items after `expires_at`.

Webhook status-update events use the same verification `id` and follow the same semantics as single-email verification (`POST /verify`).

Common verification events:

- `verify.created`
- `verify.check.first.no_bounce`
- `verify.check.second.no_bounce`
- `verify.check.first.error`
- `verify.check.second.error`
- `verify.invalid`

## Webhook Receiver Example (Go)

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type VerificationWebhookPayload struct {
	Event            string `json:"event"`
	ID               string `json:"id"`
	Email            string `json:"email"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	Source           string `json:"source"`
	UserID           string `json:"user_id"`
	CheckCount       int    `json:"check_count"`
	Finalized        bool   `json:"finalized"`
	CheckedAt        int64  `json:"checked_at"`
	Confidence       string `json:"confidence"`
	Deterministic    bool   `json:"deterministic"`
	ReasonCode       string `json:"reason_code"`
	VerificationPath string `json:"verification_path"`
	SignalSummary    string `json:"signal_summary"`
	ExpiresAt        int64  `json:"expires_at"`
}

func verificationWebhookHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload VerificationWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Idempotency recommendation: upsert by payload.ID and ignore stale updates.
	log.Printf("event=%s id=%s status=%s finalized=%v", payload.Event, payload.ID, payload.Status, payload.Finalized)

	w.WriteHeader(http.StatusNoContent)
}
```

For full webhook setup, payload details, and endpoint examples, see [`docs/webhook-api.md`](./webhook-api.md).
