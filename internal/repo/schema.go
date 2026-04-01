package repo

import (
	"context"
	"fmt"
)

func (r *Repository) initSchema(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL DEFAULT '',
	api_key TEXT NOT NULL UNIQUE,
	webhook_url TEXT NOT NULL DEFAULT '',
	is_superuser BOOLEAN NOT NULL DEFAULT FALSE,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_api_key ON users (api_key);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE TABLE IF NOT EXISTS verifications (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL,
	user_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	message TEXT NOT NULL,
	source TEXT NOT NULL,
	probe_token TEXT NOT NULL DEFAULT '',
	smtp_account_id TEXT NOT NULL DEFAULT '',
	confidence TEXT NOT NULL DEFAULT 'low',
	deterministic BOOLEAN NOT NULL DEFAULT FALSE,
	reason_code TEXT NOT NULL DEFAULT '',
	verification_path TEXT NOT NULL DEFAULT '',
	signal_summary TEXT NOT NULL DEFAULT '',
	expires_at BIGINT NOT NULL DEFAULT 0,
	check_count INTEGER NOT NULL DEFAULT 0,
	finalized BOOLEAN NOT NULL DEFAULT FALSE,
	first_checked_at BIGINT NOT NULL,
	last_checked_at BIGINT NOT NULL,
	next_check_at BIGINT NOT NULL DEFAULT 0,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL,
	UNIQUE(email, user_id)
);

CREATE INDEX IF NOT EXISTS idx_verifications_next_check ON verifications (next_check_at, finalized);
CREATE INDEX IF NOT EXISTS idx_verifications_user_id ON verifications (user_id);

CREATE TABLE IF NOT EXISTS verification_events (
	id BIGSERIAL PRIMARY KEY,
	verification_id TEXT NOT NULL,
	event_type TEXT NOT NULL,
	status TEXT NOT NULL,
	message TEXT NOT NULL,
	created_at BIGINT NOT NULL,
	FOREIGN KEY (verification_id) REFERENCES verifications(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS smtp_accounts (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL DEFAULT '',
	host TEXT NOT NULL,
	port INTEGER NOT NULL,
	username TEXT NOT NULL,
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
	updated_at BIGINT NOT NULL,
	UNIQUE(username, user_id)
);

CREATE INDEX IF NOT EXISTS idx_smtp_accounts_active_usage ON smtp_accounts (active, sent_today);
CREATE INDEX IF NOT EXISTS idx_smtp_accounts_user_id ON smtp_accounts (user_id);

CREATE TABLE IF NOT EXISTS email_templates (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	subject_template TEXT NOT NULL,
	body_template TEXT NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at BIGINT NOT NULL,
	updated_at BIGINT NOT NULL,
	UNIQUE(name, user_id)
);

CREATE INDEX IF NOT EXISTS idx_email_templates_active ON email_templates (active, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_templates_user_id ON email_templates (user_id);

-- Migration: Add user_id columns if they don't exist
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE smtp_accounts ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE email_templates ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS smtp_account_id TEXT NOT NULL DEFAULT '';
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS confidence TEXT NOT NULL DEFAULT 'low';
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS deterministic BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS reason_code TEXT NOT NULL DEFAULT '';
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS verification_path TEXT NOT NULL DEFAULT '';
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS signal_summary TEXT NOT NULL DEFAULT '';
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS expires_at BIGINT NOT NULL DEFAULT 0;
ALTER TABLE smtp_accounts ADD COLUMN IF NOT EXISTS imap_host TEXT NOT NULL DEFAULT '';
ALTER TABLE smtp_accounts ADD COLUMN IF NOT EXISTS imap_port INTEGER NOT NULL DEFAULT 993;
ALTER TABLE smtp_accounts ADD COLUMN IF NOT EXISTS imap_mailbox TEXT NOT NULL DEFAULT 'INBOX';

-- Migration: Add password_hash and is_superuser to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_superuser BOOLEAN NOT NULL DEFAULT FALSE;

-- Drop old unique constraints if they exist and create new ones
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'verifications_email_key') THEN
        ALTER TABLE verifications DROP CONSTRAINT verifications_email_key;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'smtp_accounts_username_key') THEN
        ALTER TABLE smtp_accounts DROP CONSTRAINT smtp_accounts_username_key;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'email_templates_name_key') THEN
        ALTER TABLE email_templates DROP CONSTRAINT email_templates_name_key;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Migration: ensure verification events cascade delete with parent verification
DO $$ BEGIN
	IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'verification_events_verification_id_fkey') THEN
		ALTER TABLE verification_events DROP CONSTRAINT verification_events_verification_id_fkey;
	END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

DO $$ BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'verification_events_verification_id_fkey') THEN
		ALTER TABLE verification_events
			ADD CONSTRAINT verification_events_verification_id_fkey
			FOREIGN KEY (verification_id) REFERENCES verifications(id) ON DELETE CASCADE;
	END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
`

	if _, err := r.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	return nil
}
