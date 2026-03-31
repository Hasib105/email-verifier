package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"email-verifier-api/internal/store"
)

func (r *Repository) CreateEmailTemplate(ctx context.Context, input store.EmailTemplateInput) (*store.EmailTemplate, error) {
	now := time.Now().Unix()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if input.Active {
		if _, err := tx.ExecContext(ctx, `UPDATE email_templates SET active = FALSE, updated_at = $1 WHERE active = TRUE AND user_id = $2`, now, input.UserID); err != nil {
			return nil, fmt.Errorf("deactivate old templates: %w", err)
		}
	}

	query := `
INSERT INTO email_templates (id, user_id, name, subject_template, body_template, active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, name, subject_template, body_template, active, created_at, updated_at
`
	var rec store.EmailTemplate
	if err := tx.GetContext(ctx, &rec, query, input.ID, input.UserID, input.Name, input.SubjectTemplate, input.BodyTemplate, input.Active, now, now); err != nil {
		return nil, fmt.Errorf("create email template: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return &rec, nil
}

func (r *Repository) ListEmailTemplates(ctx context.Context) ([]store.EmailTemplate, error) {
	records := []store.EmailTemplate{}
	query := `
SELECT id, user_id, name, subject_template, body_template, active, created_at, updated_at
FROM email_templates
ORDER BY active DESC, updated_at DESC
`
	if err := r.db.SelectContext(ctx, &records, query); err != nil {
		return nil, fmt.Errorf("list email templates: %w", err)
	}
	return records, nil
}

func (r *Repository) ListEmailTemplatesByUser(ctx context.Context, userID string) ([]store.EmailTemplate, error) {
	records := []store.EmailTemplate{}
	query := `
SELECT id, user_id, name, subject_template, body_template, active, created_at, updated_at
FROM email_templates
WHERE user_id = $1
ORDER BY active DESC, updated_at DESC
`
	if err := r.db.SelectContext(ctx, &records, query, userID); err != nil {
		return nil, fmt.Errorf("list email templates by user: %w", err)
	}
	return records, nil
}

func (r *Repository) GetActiveEmailTemplate(ctx context.Context) (*store.EmailTemplate, error) {
	var rec store.EmailTemplate
	query := `
SELECT id, user_id, name, subject_template, body_template, active, created_at, updated_at
FROM email_templates
WHERE active = TRUE
ORDER BY updated_at DESC
LIMIT 1
`
	if err := r.db.GetContext(ctx, &rec, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active email template: %w", err)
	}
	return &rec, nil
}

func (r *Repository) GetActiveEmailTemplateByUser(ctx context.Context, userID string) (*store.EmailTemplate, error) {
	var rec store.EmailTemplate
	query := `
SELECT id, user_id, name, subject_template, body_template, active, created_at, updated_at
FROM email_templates
WHERE active = TRUE AND user_id = $1
ORDER BY updated_at DESC
LIMIT 1
`
	if err := r.db.GetContext(ctx, &rec, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active email template by user: %w", err)
	}
	return &rec, nil
}

// GetRotatingEmailTemplate returns a template using round-robin rotation
// It uses the count parameter to determine which template to use
func (r *Repository) GetRotatingEmailTemplate(ctx context.Context, userID string, rotationIndex int) (*store.EmailTemplate, error) {
	templates := []store.EmailTemplate{}
	query := `
SELECT id, user_id, name, subject_template, body_template, active, created_at, updated_at
FROM email_templates
WHERE user_id = $1
ORDER BY created_at ASC
`
	if err := r.db.SelectContext(ctx, &templates, query, userID); err != nil {
		return nil, fmt.Errorf("list email templates for rotation: %w", err)
	}

	if len(templates) == 0 {
		return nil, nil
	}

	// Round-robin selection
	index := rotationIndex % len(templates)
	return &templates[index], nil
}

func (r *Repository) GetEmailTemplateByID(ctx context.Context, id string) (*store.EmailTemplate, error) {
	var rec store.EmailTemplate
	query := `
SELECT id, user_id, name, subject_template, body_template, active, created_at, updated_at
FROM email_templates
WHERE id = $1
`
	if err := r.db.GetContext(ctx, &rec, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get email template by id: %w", err)
	}
	return &rec, nil
}

func (r *Repository) UpdateEmailTemplate(ctx context.Context, id string, input store.EmailTemplateInput) (*store.EmailTemplate, error) {
	now := time.Now().Unix()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if input.Active {
		if _, err := tx.ExecContext(ctx, `UPDATE email_templates SET active = FALSE, updated_at = $1 WHERE active = TRUE AND user_id = $2 AND id != $3`, now, input.UserID, id); err != nil {
			return nil, fmt.Errorf("deactivate old templates: %w", err)
		}
	}

	query := `
UPDATE email_templates
SET name = $2, subject_template = $3, body_template = $4, active = $5, updated_at = $6
WHERE id = $1
RETURNING id, user_id, name, subject_template, body_template, active, created_at, updated_at
`
	var rec store.EmailTemplate
	if err := tx.GetContext(ctx, &rec, query, id, input.Name, input.SubjectTemplate, input.BodyTemplate, input.Active, now); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update email template: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return &rec, nil
}

func (r *Repository) DeleteEmailTemplate(ctx context.Context, id string) error {
	query := `DELETE FROM email_templates WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete email template: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("email template not found")
	}
	return nil
}
