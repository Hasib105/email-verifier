package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"email-verifier-api/internal/store"

	"github.com/google/uuid"
)

func (r *Repository) GetByEmailAndUser(ctx context.Context, email, userID string) (*store.VerificationRecord, error) {
	var rec store.VerificationRecord
	query := `
SELECT id, email, domain, user_id, classification, confidence_score, risk_level, deterministic, state,
       reason_codes, protocol_summary, enrichment_summary, expires_at, last_verified_at, last_enriched_at,
       created_at, updated_at
FROM verifications
WHERE email = $1 AND user_id = $2
`

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
	id, email, domain, user_id, classification, confidence_score, risk_level, deterministic, state,
	reason_codes, protocol_summary, enrichment_summary, expires_at, last_verified_at, last_enriched_at,
	status, message, source, finalized, first_checked_at, last_checked_at, next_check_at, created_at, updated_at
) VALUES (
	:id, :email, :domain, :user_id, :classification, :confidence_score, :risk_level, :deterministic, :state,
	:reason_codes, :protocol_summary, :enrichment_summary, :expires_at, :last_verified_at, :last_enriched_at,
	:classification, :protocol_summary, 'v2-callout', TRUE, :last_verified_at, :last_verified_at, 0, :created_at, :updated_at
)
ON CONFLICT(email, user_id) DO UPDATE SET
	domain = excluded.domain,
	classification = excluded.classification,
	confidence_score = excluded.confidence_score,
	risk_level = excluded.risk_level,
	deterministic = excluded.deterministic,
	state = excluded.state,
	reason_codes = excluded.reason_codes,
	protocol_summary = excluded.protocol_summary,
	enrichment_summary = excluded.enrichment_summary,
	expires_at = excluded.expires_at,
	last_verified_at = excluded.last_verified_at,
	last_enriched_at = excluded.last_enriched_at,
	status = excluded.status,
	message = excluded.message,
	source = excluded.source,
	finalized = excluded.finalized,
	first_checked_at = COALESCE(NULLIF(verifications.first_checked_at, 0), excluded.first_checked_at),
	last_checked_at = excluded.last_checked_at,
	next_check_at = 0,
	updated_at = excluded.updated_at
`

	if _, err := r.db.NamedExecContext(ctx, query, rec); err != nil {
		return fmt.Errorf("upsert verification: %w", err)
	}
	return nil
}

func (r *Repository) UpdateVerificationEnrichment(ctx context.Context, id string, confidence int, riskLevel, state, summary string) error {
	query := `
UPDATE verifications
SET confidence_score = $2,
    risk_level = $3,
    state = $4,
    enrichment_summary = $5,
    last_enriched_at = $6,
    updated_at = $6
WHERE id = $1
`
	now := time.Now().Unix()
	if _, err := r.db.ExecContext(ctx, query, id, confidence, riskLevel, state, summary, now); err != nil {
		return fmt.Errorf("update verification enrichment: %w", err)
	}
	return nil
}

func (r *Repository) ReplaceEnrichmentEvidence(ctx context.Context, verificationID string, evidence []store.EnrichmentEvidence) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin evidence transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM enrichment_evidence WHERE verification_id = $1`, verificationID); err != nil {
		return fmt.Errorf("delete enrichment evidence: %w", err)
	}

	query := `
INSERT INTO enrichment_evidence (id, verification_id, source, kind, signal, weight, summary, created_at)
VALUES (:id, :verification_id, :source, :kind, :signal, :weight, :summary, :created_at)
`
	for i := range evidence {
		if evidence[i].ID == "" {
			evidence[i].ID = uuid.NewString()
		}
		if evidence[i].VerificationID == "" {
			evidence[i].VerificationID = verificationID
		}
		if evidence[i].CreatedAt == 0 {
			evidence[i].CreatedAt = time.Now().Unix()
		}
		if _, err := tx.NamedExecContext(ctx, query, evidence[i]); err != nil {
			return fmt.Errorf("insert enrichment evidence: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit evidence transaction: %w", err)
	}
	return nil
}

func (r *Repository) ListEnrichmentEvidence(ctx context.Context, verificationID string) ([]store.EnrichmentEvidence, error) {
	items := []store.EnrichmentEvidence{}
	query := `
SELECT id, verification_id, source, kind, signal, weight, summary, created_at
FROM enrichment_evidence
WHERE verification_id = $1
ORDER BY created_at DESC
`
	if err := r.db.SelectContext(ctx, &items, query, verificationID); err != nil {
		return nil, fmt.Errorf("list enrichment evidence: %w", err)
	}
	return items, nil
}

func (r *Repository) AddCalloutAttempts(ctx context.Context, verificationID string, attempts []store.VerificationCalloutAttempt) error {
	if len(attempts) == 0 {
		return nil
	}

	query := `
INSERT INTO verification_callouts (verification_id, smtp_host, smtp_port, stage, recipient, outcome, smtp_code, smtp_message, duration_ms, created_at)
VALUES (:verification_id, :smtp_host, :smtp_port, :stage, :recipient, :outcome, :smtp_code, :smtp_message, :duration_ms, :created_at)
`
	now := time.Now().Unix()
	for i := range attempts {
		attempts[i].VerificationID = verificationID
		if attempts[i].CreatedAt == 0 {
			attempts[i].CreatedAt = now
		}
		if attempts[i].SMTPPort == 0 {
			attempts[i].SMTPPort = 25
		}
		if _, err := r.db.NamedExecContext(ctx, query, attempts[i]); err != nil {
			return fmt.Errorf("insert verification callout: %w", err)
		}
	}
	return nil
}

func (r *Repository) ListCalloutAttempts(ctx context.Context, verificationID string) ([]store.VerificationCalloutAttempt, error) {
	items := []store.VerificationCalloutAttempt{}
	query := `
SELECT id, verification_id, smtp_host, smtp_port, stage, recipient, outcome, smtp_code, smtp_message, duration_ms, created_at
FROM verification_callouts
WHERE verification_id = $1
ORDER BY created_at ASC, id ASC
`
	if err := r.db.SelectContext(ctx, &items, query, verificationID); err != nil {
		return nil, fmt.Errorf("list verification callouts: %w", err)
	}
	return items, nil
}

func (r *Repository) GetDomainBaseline(ctx context.Context, domain, fingerprint string, nowUnix int64) (*store.DomainBaseline, error) {
	var baseline store.DomainBaseline
	query := `
SELECT domain, mx_fingerprint, classification, sample_address, smtp_host, smtp_code, smtp_message, checked_at, expires_at, created_at, updated_at
FROM domain_baselines
WHERE domain = $1 AND mx_fingerprint = $2 AND expires_at > $3
`
	if err := r.db.GetContext(ctx, &baseline, query, domain, fingerprint, nowUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get domain baseline: %w", err)
	}
	return &baseline, nil
}

func (r *Repository) UpsertDomainBaseline(ctx context.Context, baseline *store.DomainBaseline) error {
	query := `
INSERT INTO domain_baselines (
	domain, mx_fingerprint, classification, sample_address, smtp_host, smtp_code, smtp_message, checked_at, expires_at, created_at, updated_at
) VALUES (
	:domain, :mx_fingerprint, :classification, :sample_address, :smtp_host, :smtp_code, :smtp_message, :checked_at, :expires_at, :created_at, :updated_at
)
ON CONFLICT(domain, mx_fingerprint) DO UPDATE SET
	classification = excluded.classification,
	sample_address = excluded.sample_address,
	smtp_host = excluded.smtp_host,
	smtp_code = excluded.smtp_code,
	smtp_message = excluded.smtp_message,
	checked_at = excluded.checked_at,
	expires_at = excluded.expires_at,
	updated_at = excluded.updated_at
`
	if _, err := r.db.NamedExecContext(ctx, query, baseline); err != nil {
		return fmt.Errorf("upsert domain baseline: %w", err)
	}
	return nil
}

func (r *Repository) ListVerificationsByUser(ctx context.Context, userID string, limit, offset int) ([]store.VerificationRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	items := []store.VerificationRecord{}
	query := `
SELECT id, email, domain, user_id, classification, confidence_score, risk_level, deterministic, state,
       reason_codes, protocol_summary, enrichment_summary, expires_at, last_verified_at, last_enriched_at,
       created_at, updated_at
FROM verifications
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3
`
	if err := r.db.SelectContext(ctx, &items, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("list verifications by user: %w", err)
	}
	return items, nil
}

func (r *Repository) GetVerificationStats(ctx context.Context, userID string) (map[string]int, error) {
	stats := make(map[string]int)
	query := `SELECT classification, COUNT(*) FROM verifications WHERE user_id = $1 GROUP BY classification`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get verification stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var classification string
		var count int
		if err := rows.Scan(&classification, &count); err != nil {
			return nil, err
		}
		stats[classification] = count
	}
	return stats, nil
}

func (r *Repository) GetVerificationByID(ctx context.Context, id string) (*store.VerificationRecord, error) {
	var rec store.VerificationRecord
	query := `
SELECT id, email, domain, user_id, classification, confidence_score, risk_level, deterministic, state,
       reason_codes, protocol_summary, enrichment_summary, expires_at, last_verified_at, last_enriched_at,
       created_at, updated_at
FROM verifications
WHERE id = $1
`
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
	items := []store.VerificationRecord{}
	query := `
SELECT id, email, domain, user_id, classification, confidence_score, risk_level, deterministic, state,
       reason_codes, protocol_summary, enrichment_summary, expires_at, last_verified_at, last_enriched_at,
       created_at, updated_at
FROM verifications
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2
`
	if err := r.db.SelectContext(ctx, &items, query, limit, offset); err != nil {
		return nil, fmt.Errorf("list all verifications: %w", err)
	}
	return items, nil
}

func (r *Repository) DeleteVerification(ctx context.Context, id string) error {
	query := `DELETE FROM verifications WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete verification: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("verification not found")
	}
	return nil
}
