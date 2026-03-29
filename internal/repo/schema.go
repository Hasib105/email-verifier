package repo

import (
	"context"
	"fmt"
)

func (r *Repository) initSchema(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS verifications (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	status TEXT NOT NULL,
	message TEXT NOT NULL,
	source TEXT NOT NULL,
	probe_token TEXT NOT NULL DEFAULT '',
	smtp_account_id TEXT NOT NULL DEFAULT '',
	check_count INTEGER NOT NULL DEFAULT 0,
	finalized BOOLEAN NOT NULL DEFAULT FALSE,
	first_checked_at BIGINT NOT NULL,
	last_checked_at BIGINT NOT NULL,
	next_check_at BIGINT NOT NULL DEFAULT 0,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_verifications_next_check ON verifications (next_check_at, finalized);

CREATE TABLE IF NOT EXISTS verification_events (
	id BIGSERIAL PRIMARY KEY,
	verification_id TEXT NOT NULL,
	event_type TEXT NOT NULL,
	status TEXT NOT NULL,
	message TEXT NOT NULL,
	created_at BIGINT NOT NULL,
	FOREIGN KEY (verification_id) REFERENCES verifications(id)
);

CREATE TABLE IF NOT EXISTS smtp_accounts (
	id TEXT PRIMARY KEY,
	host TEXT NOT NULL,
	port INTEGER NOT NULL,
	username TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	sender TEXT NOT NULL,
	imap_host TEXT NOT NULL,
	imap_port INTEGER NOT NULL,
	imap_mailbox TEXT NOT NULL DEFAULT 'INBOX',
	daily_limit INTEGER NOT NULL DEFAULT 100,
	sent_today INTEGER NOT NULL DEFAULT 0,
	reset_date DATE NOT NULL DEFAULT CURRENT_DATE,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_smtp_accounts_active_usage ON smtp_accounts (active, sent_today);

CREATE TABLE IF NOT EXISTS email_templates (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	subject_template TEXT NOT NULL,
	body_template TEXT NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_email_templates_active ON email_templates (active, updated_at DESC);

ALTER TABLE verifications ADD COLUMN IF NOT EXISTS smtp_account_id TEXT NOT NULL DEFAULT '';
ALTER TABLE smtp_accounts ADD COLUMN IF NOT EXISTS imap_host TEXT NOT NULL DEFAULT '';
ALTER TABLE smtp_accounts ADD COLUMN IF NOT EXISTS imap_port INTEGER NOT NULL DEFAULT 993;
ALTER TABLE smtp_accounts ADD COLUMN IF NOT EXISTS imap_mailbox TEXT NOT NULL DEFAULT 'INBOX';
`

	if _, err := r.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	return nil
}
