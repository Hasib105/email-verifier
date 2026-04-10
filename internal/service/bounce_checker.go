package service

import (
	"context"
	"crypto/tls"
	"email-verifier-api/internal/serviceutil"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type IMAPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Mailbox  string
}

type IMAPBounceChecker struct {
}

func NewIMAPBounceChecker() *IMAPBounceChecker {
	return &IMAPBounceChecker{}
}

func (c *IMAPBounceChecker) HasBounce(_ context.Context, cfg IMAPConfig, targetEmail, token string) (bool, string, string, error) {
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		return false, "", "", fmt.Errorf("imap is not configured")
	}

	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	imapClient, err := client.DialTLS(address, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return false, "", "", fmt.Errorf("imap dial: %w", err)
	}
	defer imapClient.Logout()

	if err := imapClient.Login(cfg.Username, cfg.Password); err != nil {
		return false, "", "", fmt.Errorf("imap login: %w", err)
	}

	if _, err := imapClient.Select(mailbox, true); err != nil {
		return false, "", "", fmt.Errorf("imap select mailbox: %w", err)
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Now().Add(-48 * time.Hour)
	ids, err := imapClient.Search(criteria)
	if err != nil {
		return false, "", "", fmt.Errorf("imap search: %w", err)
	}
	if len(ids) == 0 {
		return false, "", "", nil
	}

	if len(ids) > 40 {
		ids = ids[len(ids)-40:]
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(ids...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}
	messages := make(chan *imap.Message, len(ids))

	errCh := make(chan error, 1)
	go func() {
		errCh <- imapClient.Fetch(seqSet, items, messages)
	}()

	lowerToken := strings.ToLower(token)
	lowerTarget := strings.ToLower(targetEmail)

	for msg := range messages {
		if msg == nil {
			continue
		}

		body := msg.GetBody(section)
		if body == nil {
			continue
		}

		raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
		if err != nil {
			continue
		}

		text := strings.ToLower(string(raw))
		subject := ""
		if msg.Envelope != nil {
			subject = msg.Envelope.Subject
		}

		if !serviceutil.ContainsBounceSignature(text) {
			continue
		}

		if lowerToken != "" && strings.Contains(text, lowerToken) {
			return true, fmt.Sprintf("Bounce detected by probe token in subject: %s", subject), "token_match", nil
		}

		if lowerToken == "" && lowerTarget != "" && strings.Contains(text, lowerTarget) {
			return true, fmt.Sprintf("Bounce detected for recipient in subject: %s", subject), "recipient_match", nil
		}

		if lowerToken == "" {
			return true, fmt.Sprintf("Generic delivery-status notification detected in subject: %s", subject), "generic_dsn", nil
		}
	}

	if fetchErr := <-errCh; fetchErr != nil {
		return false, "", "", fmt.Errorf("imap fetch: %w", fetchErr)
	}

	return false, "", "", nil
}
