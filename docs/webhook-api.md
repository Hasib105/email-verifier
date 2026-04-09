# Webhook Integration Guide

This guide explains how to receive asynchronous verification updates through webhooks.

## Why Webhooks

Some verifications start in `pending_bounce_check`, then may become provisional `valid` before finalizing after IMAP bounce checks. Webhooks let you receive updates without polling.

Webhook behavior covers both verification entry points:

- `POST /verify` (single email)
- `POST /verify/batch` (multiple emails)

If any verification status changes after the initial response, the service sends another webhook event for the same verification `id`.

## Configure Webhook URL

- Method: `PUT`
- URL: `/users/webhook`
- Header: `X-API-Key: <user-api-key>`
- Content type: `application/json`

Request:

```json
{
  "webhook_url": "https://your-system.example.com/email-verifier/webhook"
}
```

Success response:

```json
{
  "message": "Webhook URL updated successfully"
}
```

## Send Test Webhook

- Method: `POST`
- URL: `/users/webhook/test`
- Header: `X-API-Key: <user-api-key>`
- Content type: `application/json`

Request:

```json
{
  "webhook_url": "https://your-system.example.com/email-verifier/webhook"
}
```

Success response:

```json
{
  "message": "Test webhook sent successfully"
}
```

The test event uses `event = test.webhook`.

## Event Types

The service emits these verification events:

- `verify.created`
- `verify.check.first.no_bounce`
- `verify.check.second.no_bounce`
- `verify.check.first.error`
- `verify.check.second.error`
- `verify.invalid`

Typical status progression for fallback cases:

- `pending_bounce_check` -> `valid` with `finalized=false` (`verify.check.first.no_bounce`)
- `valid` -> `invalid` (`verify.invalid`)
- `valid` -> `valid` with `finalized=true` (`verify.check.second.no_bounce`)
- `pending_bounce_check` -> `error` (`verify.check.second.error`)

## Webhook Payload

Payload fields sent by the API:

- `event`
- `id`
- `email`
- `status`
- `message`
- `source`
- `user_id`
- `check_count`
- `finalized`
- `checked_at`
- `confidence`
- `deterministic`
- `reason_code`
- `verification_path`
- `signal_summary`
- `expires_at`

Example payload:

```json
{
  "event": "verify.check.second.no_bounce",
  "id": "5f7ea2f8-cf04-4888-b8d8-e50ef9040ac5",
  "email": "user@example.com",
  "status": "valid",
  "message": "no bounce observed within the configured verification window",
  "source": "smtp-probe",
  "user_id": "e72b76b2-a4db-4e0f-b70a-7b31f57a16e9",
  "check_count": 2,
  "finalized": true,
  "checked_at": 1774900800,
  "confidence": "low",
  "deterministic": false,
  "reason_code": "no_bounce_second_window",
  "verification_path": "probe_bounce",
  "signal_summary": "No bounce was observed across both check windows. This remains a heuristic signal rather than confirmed mailbox existence.",
  "expires_at": 1774987200
}
```

## Delivery Semantics

- Delivery is per user webhook URL (`users.webhook_url`) or fallback `WEBHOOK_URL` env var.
- A `2xx` response from your endpoint is considered success.
- Non-`2xx` responses are treated as failures.
- Current behavior is best-effort delivery (no built-in retry queue).

## Receiver Best Practices

- Use HTTPS and verify request origin in your edge layer.
- Treat handlers as idempotent and upsert by `id`.
- Always update your stored status by payload `id` when a newer webhook event arrives.
- Use `finalized` to decide when a result can stop changing.
- Keep a reconciliation job using `GET /verifications/:id` for rare missed deliveries.
- Re-verify records after `expires_at`.
