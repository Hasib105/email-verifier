package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"email-verifier-api/internal/store"
)

const defaultDailyLimit = 100

func (r *Repository) CreateSMTPAccount(ctx context.Context, input store.SMTPAccountInput) (*store.SMTPAccount, error) {
	if input.Port == 0 {
		input.Port = 587
	}
	if input.IMAPPort == 0 {
		input.IMAPPort = 993
	}
	if input.IMAPHost == "" {
		input.IMAPHost = input.Host
	}
	if input.IMAPMailbox == "" {
		input.IMAPMailbox = "INBOX"
	}
	if input.DailyLimit <= 0 {
		input.DailyLimit = defaultDailyLimit
	}

	now := time.Now().Unix()
	query := `
INSERT INTO smtp_accounts (id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 0, CURRENT_DATE, $12, $13, $14)
RETURNING id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
`

	var rec store.SMTPAccount
	err := r.db.GetContext(ctx, &rec, query,
		input.ID, input.UserID, input.Host, input.Port, input.Username, input.Password, input.Sender,
		input.IMAPHost, input.IMAPPort, input.IMAPMailbox, input.DailyLimit, input.Active, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create smtp account: %w", err)
	}
	return &rec, nil
}

func (r *Repository) ListSMTPAccounts(ctx context.Context) ([]store.SMTPAccount, error) {
	records := []store.SMTPAccount{}
	query := `
SELECT id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
FROM smtp_accounts
ORDER BY active DESC, sent_today ASC, created_at ASC
`
	if err := r.db.SelectContext(ctx, &records, query); err != nil {
		return nil, fmt.Errorf("list smtp accounts: %w", err)
	}
	return records, nil
}

func (r *Repository) ListSMTPAccountsByUser(ctx context.Context, userID string) ([]store.SMTPAccount, error) {
	records := []store.SMTPAccount{}
	query := `
SELECT id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
FROM smtp_accounts
WHERE user_id = $1
ORDER BY active DESC, sent_today ASC, created_at ASC
`
	if err := r.db.SelectContext(ctx, &records, query, userID); err != nil {
		return nil, fmt.Errorf("list smtp accounts by user: %w", err)
	}
	return records, nil
}

func (r *Repository) AcquireSMTPAccountForSend(ctx context.Context) (*store.SMTPAccount, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE smtp_accounts SET sent_today = 0, reset_date = CURRENT_DATE, updated_at = $1 WHERE reset_date < CURRENT_DATE`, time.Now().Unix()); err != nil {
		return nil, fmt.Errorf("reset smtp counters: %w", err)
	}

	var acc store.SMTPAccount
	selectQuery := `
SELECT id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
FROM smtp_accounts
WHERE active = TRUE AND sent_today < daily_limit
ORDER BY sent_today ASC, created_at ASC
FOR UPDATE SKIP LOCKED
LIMIT 1
`
	if err := tx.GetContext(ctx, &acc, selectQuery); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select smtp account: %w", err)
	}

	updateQuery := `
UPDATE smtp_accounts
SET sent_today = sent_today + 1, updated_at = $2
WHERE id = $1
RETURNING id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
`
	if err := tx.GetContext(ctx, &acc, updateQuery, acc.ID, time.Now().Unix()); err != nil {
		return nil, fmt.Errorf("increment smtp usage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return &acc, nil
}

func (r *Repository) AcquireSMTPAccountForSendByUser(ctx context.Context, userID string) (*store.SMTPAccount, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE smtp_accounts SET sent_today = 0, reset_date = CURRENT_DATE, updated_at = $1 WHERE reset_date < CURRENT_DATE AND user_id = $2`, time.Now().Unix(), userID); err != nil {
		return nil, fmt.Errorf("reset smtp counters: %w", err)
	}

	var acc store.SMTPAccount
	selectQuery := `
SELECT id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
FROM smtp_accounts
WHERE active = TRUE AND sent_today < daily_limit AND user_id = $1
ORDER BY sent_today ASC, created_at ASC
FOR UPDATE SKIP LOCKED
LIMIT 1
`
	if err := tx.GetContext(ctx, &acc, selectQuery, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select smtp account: %w", err)
	}

	updateQuery := `
UPDATE smtp_accounts
SET sent_today = sent_today + 1, updated_at = $2
WHERE id = $1
RETURNING id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
`
	if err := tx.GetContext(ctx, &acc, updateQuery, acc.ID, time.Now().Unix()); err != nil {
		return nil, fmt.Errorf("increment smtp usage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return &acc, nil
}

func (r *Repository) GetSMTPAccountByID(ctx context.Context, id string) (*store.SMTPAccount, error) {
	var rec store.SMTPAccount
	query := `
SELECT id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
FROM smtp_accounts
WHERE id = $1
`
	if err := r.db.GetContext(ctx, &rec, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get smtp account by id: %w", err)
	}
	return &rec, nil
}

func (r *Repository) UpdateSMTPAccount(ctx context.Context, id string, input store.SMTPAccountInput) (*store.SMTPAccount, error) {
	now := time.Now().Unix()
	query := `
UPDATE smtp_accounts
SET host = $2, port = $3, username = $4, password = $5, sender = $6, imap_host = $7, imap_port = $8, imap_mailbox = $9, daily_limit = $10, active = $11, updated_at = $12
WHERE id = $1
RETURNING id, user_id, host, port, username, password, sender, imap_host, imap_port, imap_mailbox, daily_limit, sent_today, reset_date, active, created_at, updated_at
`
	var rec store.SMTPAccount
	err := r.db.GetContext(ctx, &rec, query,
		id, input.Host, input.Port, input.Username, input.Password, input.Sender, input.IMAPHost, input.IMAPPort, input.IMAPMailbox, input.DailyLimit, input.Active, now,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update smtp account: %w", err)
	}
	return &rec, nil
}

func (r *Repository) DeleteSMTPAccount(ctx context.Context, id string) error {
	query := `DELETE FROM smtp_accounts WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete smtp account: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("smtp account not found")
	}
	return nil
}
