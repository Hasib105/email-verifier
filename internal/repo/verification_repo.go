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

func (r *Repository) ListVerificationsByUser(ctx context.Context, userID string, limit, offset int) ([]store.VerificationRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	records := []store.VerificationRecord{}
	query := `SELECT id, email, user_id, status, message, source, probe_token, smtp_account_id, check_count, finalized,
	first_checked_at, last_checked_at, next_check_at, created_at, updated_at
	FROM verifications
	WHERE user_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`

	if err := r.db.SelectContext(ctx, &records, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("list verifications by user: %w", err)
	}
	return records, nil
}

func (r *Repository) CountVerificationsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM verifications WHERE user_id = $1`
	if err := r.db.GetContext(ctx, &count, query, userID); err != nil {
		return 0, fmt.Errorf("count verifications by user: %w", err)
	}
	return count, nil
}

func (r *Repository) GetVerificationStats(ctx context.Context, userID string) (map[string]int, error) {
	stats := make(map[string]int)
	query := `SELECT status, COUNT(*) as count FROM verifications WHERE user_id = $1 GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get verification stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}
	return stats, nil
}

func (r *Repository) GetVerificationByID(ctx context.Context, id string) (*store.VerificationRecord, error) {
	var rec store.VerificationRecord
	query := `SELECT id, email, user_id, status, message, source, probe_token, smtp_account_id, check_count, finalized,
	first_checked_at, last_checked_at, next_check_at, created_at, updated_at
	FROM verifications WHERE id = $1`

	if err := r.db.GetContext(ctx, &rec, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get verification by id: %w", err)
	}
	return &rec, nil
}

func (r *Repository) ListAllVerifications(ctx context.Context, limit, offset int) ([]store.VerificationRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	records := []store.VerificationRecord{}
	query := `SELECT id, email, user_id, status, message, source, probe_token, smtp_account_id, check_count, finalized,
	first_checked_at, last_checked_at, next_check_at, created_at, updated_at
	FROM verifications
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`

	if err := r.db.SelectContext(ctx, &records, query, limit, offset); err != nil {
		return nil, fmt.Errorf("list all verifications: %w", err)
	}
	return records, nil
}

func (r *Repository) DeleteVerification(ctx context.Context, id string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete verification transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM verification_events WHERE verification_id = $1`, id); err != nil {
		return fmt.Errorf("delete verification events: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM verifications WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete verification: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete verification transaction: %w", err)
	}

	return nil
}
