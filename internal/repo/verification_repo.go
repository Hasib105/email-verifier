package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"email-verifier-api/internal/store"
)

func (r *Repository) GetByEmail(ctx context.Context, email string) (*store.VerificationRecord, error) {
	var rec store.VerificationRecord
	query := `SELECT id, email, user_id, status, message, source, probe_token, smtp_account_id, check_count, finalized,
	first_checked_at, last_checked_at, next_check_at, created_at, updated_at
	FROM verifications WHERE email = $1`

	if err := r.db.GetContext(ctx, &rec, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get verification by email: %w", err)
	}
	return &rec, nil
}

func (r *Repository) GetByEmailAndUser(ctx context.Context, email, userID string) (*store.VerificationRecord, error) {
	var rec store.VerificationRecord
	query := `SELECT id, email, user_id, status, message, source, probe_token, smtp_account_id, check_count, finalized,
	first_checked_at, last_checked_at, next_check_at, created_at, updated_at
	FROM verifications WHERE email = $1 AND user_id = $2`

	if err := r.db.GetContext(ctx, &rec, query, email, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get verification by email and user: %w", err)
	}
	return &rec, nil
}

func (r *Repository) UpsertVerification(ctx context.Context, rec *store.VerificationRecord) error {
	query := `
INSERT INTO verifications (
	id, email, user_id, status, message, source, probe_token, smtp_account_id, check_count, finalized,
	first_checked_at, last_checked_at, next_check_at, created_at, updated_at
) VALUES (
	:id, :email, :user_id, :status, :message, :source, :probe_token, :smtp_account_id, :check_count, :finalized,
	:first_checked_at, :last_checked_at, :next_check_at, :created_at, :updated_at
)
ON CONFLICT(email, user_id) DO UPDATE SET
	status = excluded.status,
	message = excluded.message,
	source = excluded.source,
	probe_token = excluded.probe_token,
	smtp_account_id = excluded.smtp_account_id,
	check_count = excluded.check_count,
	finalized = excluded.finalized,
	last_checked_at = excluded.last_checked_at,
	next_check_at = excluded.next_check_at,
	updated_at = excluded.updated_at
`

	if _, err := r.db.NamedExecContext(ctx, query, rec); err != nil {
		return fmt.Errorf("upsert verification: %w", err)
	}
	return nil
}

func (r *Repository) AddEvent(ctx context.Context, verificationID, eventType, status, message string) error {
	query := `INSERT INTO verification_events (verification_id, event_type, status, message, created_at) VALUES ($1, $2, $3, $4, $5)`
	if _, err := r.db.ExecContext(ctx, query, verificationID, eventType, status, message, time.Now().Unix()); err != nil {
		return fmt.Errorf("insert verification event: %w", err)
	}
	return nil
}

func (r *Repository) ListDueChecks(ctx context.Context, nowUnix int64, limit int) ([]store.VerificationRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	records := []store.VerificationRecord{}
	query := `SELECT id, email, user_id, status, message, source, probe_token, smtp_account_id, check_count, finalized,
	first_checked_at, last_checked_at, next_check_at, created_at, updated_at
	FROM verifications
	WHERE finalized = FALSE AND next_check_at > 0 AND next_check_at <= $1
	ORDER BY next_check_at ASC
	LIMIT $2`

	if err := r.db.SelectContext(ctx, &records, query, nowUnix, limit); err != nil {
		return nil, fmt.Errorf("list due checks: %w", err)
	}
	return records, nil
}
