# Batch Verification API Guide

This guide documents the V2 verification API for backend integrations.

## Endpoint

- Method: `POST`
- URL: `/verifications/batch`
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
- Emails are verified independently, but same-domain requests may share a cached domain baseline probe.

## Response Body

```json
{
  "total": 3,
  "accepted": 3,
  "items": [
    {
      "id": "2f06116d-4f3e-4f76-b671-71888fadb5f4",
      "email": "alice@example.com",
      "domain": "example.com",
      "classification": "deliverable",
      "confidence_score": 92,
      "risk_level": "low",
      "deterministic": true,
      "state": "completed",
      "reason_codes": ["recipient_accepted", "control_recipient_rejected"],
      "protocol_summary": "Recipient accepted by MX and domain control recipient was rejected.",
      "enrichment_summary": "",
      "cached": false
    },
    {
      "id": "f1d4f67e-54d4-437e-a5d1-3f89c53f6ff9",
      "email": "not-an-email",
      "domain": "",
      "classification": "undeliverable",
      "confidence_score": 100,
      "risk_level": "high",
      "deterministic": true,
      "state": "completed",
      "reason_codes": ["syntax_invalid"],
      "protocol_summary": "Email address failed syntax validation.",
      "enrichment_summary": "",
      "cached": false
    }
  ]
}
```

## Classification Model

- `deliverable`: recipient accepted and the same-domain control recipient was rejected.
- `undeliverable`: syntax, DNS, or SMTP returned a hard failure.
- `accept_all`: recipient accepted and the domain also accepted a random control recipient.
- `unknown`: tempfail, policy block, anti-abuse filtering, or inconclusive control-baseline behavior.

## State Model

- `completed`: protocol result is final for now; enrichment is either unnecessary or already finished.
- `enriching`: protocol result is available, and asynchronous enrichment is still running.

## Error Responses

- `401 Unauthorized`: missing or invalid API key.
- `400 Bad Request`: invalid JSON, empty list, or batch size over 1000.

Example:

```json
{
  "error": "batch limit exceeded: max 1000 emails per request"
}
```

## Retrieval APIs

- `GET /verifications?limit=50&offset=0`
- `GET /verifications/{id}`
- `GET /verifications/stats`

The detail endpoint returns the verification record plus:

- `evidence[]` for enrichment signals
- `callouts[]` for SMTP trace history

## Recommended Integration Pattern

1. Chunk large input sets into batches of at most 1000 emails.
2. Submit each chunk to `POST /verifications/batch`.
3. Use `classification`, `deterministic`, and `reason_codes` for immediate decisions.
4. For `accept_all` and `unknown`, poll `GET /verifications/{id}` if you want enrichment evidence before making a final business decision.
5. Cache your own downstream decisioning by verification `id` and `expires_at`.

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
	ID                string   `json:"id"`
	Email             string   `json:"email"`
	Domain            string   `json:"domain"`
	Classification    string   `json:"classification"`
	ConfidenceScore   int      `json:"confidence_score"`
	RiskLevel         string   `json:"risk_level"`
	Deterministic     bool     `json:"deterministic"`
	State             string   `json:"state"`
	ReasonCodes       []string `json:"reason_codes"`
	ProtocolSummary   string   `json:"protocol_summary"`
	EnrichmentSummary string   `json:"enrichment_summary"`
	Cached            bool     `json:"cached"`
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

func (c *Client) VerifyBatch(emails []string) (*BatchResponse, error) {
	var out BatchResponse
	if err := c.doJSON(http.MethodPost, "/verifications/batch", BatchRequest{Emails: emails}, &out); err != nil {
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
```
