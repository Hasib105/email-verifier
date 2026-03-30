package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"email-verifier-api/internal/store"
)

func (r *Repository) CreateUser(ctx context.Context, input store.UserInput) (*store.User, error) {
	now := time.Now().Unix()
	query := `
INSERT INTO users (id, name, email, api_key, webhook_url, active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, name, email, api_key, webhook_url, active, created_at, updated_at
`
	var rec store.User
	err := r.db.GetContext(ctx, &rec, query,
		input.ID, input.Name, input.Email, input.APIKey, input.WebhookURL, input.Active, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &rec, nil
}

func (r *Repository) GetUserByAPIKey(ctx context.Context, apiKey string) (*store.User, error) {
	var rec store.User
	query := `
SELECT id, name, email, api_key, webhook_url, active, created_at, updated_at
FROM users
WHERE api_key = $1 AND active = TRUE
`
	if err := r.db.GetContext(ctx, &rec, query, apiKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by api key: %w", err)
	}
	return &rec, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	var rec store.User
	query := `
SELECT id, name, email, api_key, webhook_url, active, created_at, updated_at
FROM users
WHERE email = $1
`
	if err := r.db.GetContext(ctx, &rec, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &rec, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	var rec store.User
	query := `
SELECT id, name, email, api_key, webhook_url, active, created_at, updated_at
FROM users
WHERE id = $1
`
	if err := r.db.GetContext(ctx, &rec, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &rec, nil
}

func (r *Repository) ListUsers(ctx context.Context) ([]store.User, error) {
	records := []store.User{}
	query := `
SELECT id, name, email, api_key, webhook_url, active, created_at, updated_at
FROM users
ORDER BY created_at DESC
`
	if err := r.db.SelectContext(ctx, &records, query); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return records, nil
}

func (r *Repository) UpdateUserWebhook(ctx context.Context, userID, webhookURL string) error {
	query := `UPDATE users SET webhook_url = $2, updated_at = $3 WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, userID, webhookURL, time.Now().Unix()); err != nil {
		return fmt.Errorf("update user webhook: %w", err)
	}
	return nil
}
