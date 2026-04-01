# Batch Verify API Integration Guide

This guide shows how other backend services can call the batch endpoint safely.

## Endpoint

- Method: `POST`
- URL: `/verify/batch`
- Auth header: `X-API-Key: <user-api-key>`
- Content type: `application/json`

## Request Body

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

- Minimum 1 email.
- Maximum 1000 emails per request.
- Emails are processed independently.

## Response Body

```json
{
  "total": 3,
  "accepted": 3,
  "items": [
    {
      "id": "2f06116d-4f3e-4f76-b671-71888fadb5f4",
      "email": "alice@example.com",
      "status": "valid",
      "message": "250 Accepted",
      "source": "direct-smtp-check",
      "cached": false,
      "finalized": true
    },
    {
      "id": "f1d4f67e-54d4-437e-a5d1-3f89c53f6ff9",
      "email": "not-an-email",
      "status": "invalid",
      "message": "invalid syntax",
      "source": "direct-smtp-check",
      "cached": false,
      "finalized": true
    }
  ]
}
```

Notes:

- `accepted` is the number of items processed by the API, not the number of `valid` emails.
- Use `items[*].status` for final business decisions.

## Error Responses

- `401 Unauthorized`: missing or invalid API key.
- `400 Bad Request`: invalid JSON, empty list, or batch size over 1000.

Example 400:

```json
{
  "error": "batch limit exceeded: max 1000 emails per request"
}
```

## Service-to-Service Implementation Pattern

1. Split your source list into chunks of at most 1000.
2. Send each chunk to `/verify/batch`.
3. Retry transient failures (`5xx`, timeout, connection reset) with exponential backoff.
4. Do not retry `4xx` except token refresh/auth correction.
5. Persist results using `id` and `email` from each item.

## Go Integration Only

This section contains only Go examples for:

- sending batch verification requests,
- pulling verification results,
- setting webhook URL,
- testing webhook delivery.

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
  ID          string `json:"id"`
  Email       string `json:"email"`
  Status      string `json:"status"`
  Message     string `json:"message"`
  Source      string `json:"source"`
  Cached      bool   `json:"cached"`
  Finalized   bool   `json:"finalized"`
  NextCheckAt int64  `json:"next_check_at,omitempty"`
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
    if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
      return err
    }
  }

  return nil
}

// 1) Submit batch request
func (c *Client) VerifyBatch(emails []string) (*BatchResponse, error) {
  var out BatchResponse
  err := c.doJSON(http.MethodPost, "/verify/batch", BatchRequest{Emails: emails}, &out)
  if err != nil {
    return nil, err
  }
  return &out, nil
}

// 2) Pull one verification result by ID
func (c *Client) GetVerificationByID(id string) (*VerifyItem, error) {
  var out VerifyItem
  err := c.doJSON(http.MethodGet, "/verifications/"+id, nil, &out)
  if err != nil {
    return nil, err
  }
  return &out, nil
}

// 3) Pull paginated verification results
func (c *Client) ListVerifications(limit, offset int) (*ListVerificationsResponse, error) {
  q := url.Values{}
  q.Set("limit", fmt.Sprintf("%d", limit))
  q.Set("offset", fmt.Sprintf("%d", offset))

  var out ListVerificationsResponse
  err := c.doJSON(http.MethodGet, "/verifications?"+q.Encode(), nil, &out)
  if err != nil {
    return nil, err
  }
  return &out, nil
}

// 4) Optional polling helper for pending_bounce_check items
func (c *Client) WaitUntilFinalized(id string, interval time.Duration, maxAttempts int) (*VerifyItem, error) {
  for i := 0; i < maxAttempts; i++ {
    item, err := c.GetVerificationByID(id)
    if err != nil {
      return nil, err
    }
    if item.Finalized {
      return item, nil
    }
    time.Sleep(interval)
  }
  return nil, fmt.Errorf("verification %s not finalized after %d attempts", id, maxAttempts)
}

// 5) Set webhook URL for this API key owner
func (c *Client) UpdateWebhook(webhookURL string) error {
  return c.doJSON(http.MethodPut, "/users/webhook", map[string]string{
    "webhook_url": webhookURL,
  }, nil)
}

// 6) Test webhook delivery immediately
func (c *Client) TestWebhook(webhookURL string) error {
  return c.doJSON(http.MethodPost, "/users/webhook/test", map[string]string{
    "webhook_url": webhookURL,
  }, nil)
}
```

## How To Pull Results

1. Call `/verify/batch` and store each `items[i].id`.
2. If `items[i].finalized` is `true`, use the result immediately.
3. If `items[i].status` is `pending_bounce_check`, poll `/verifications/{id}` until `finalized=true`.
4. For dashboard/report pages, use `/verifications?limit=50&offset=0`.

## How To Set Webhook

1. Call `PUT /users/webhook` with:

```json
{
  "webhook_url": "https://your-service.example.com/email-events"
}
```

2. Call `POST /users/webhook/test` with the same payload to verify receiver availability.

## Recommended Flow In Go Service

1. Initialize client with user API key.
2. Set webhook once using `UpdateWebhook`.
3. Submit batches using `VerifyBatch`.
4. Process immediate finalized items.
5. Poll only pending items with `WaitUntilFinalized`.
