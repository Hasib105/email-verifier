package service

import (
	"context"
	"crypto/tls"
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

func (c *IMAPBounceChecker) HasBounce(_ context.Context, cfg IMAPConfig, targetEmail, token string) (bool, string, error) {
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		return false, "", fmt.Errorf("imap is not configured")
	}

	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	imapClient, err := client.DialTLS(address, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return false, "", fmt.Errorf("imap dial: %w", err)
	}
	defer imapClient.Logout()

	if err := imapClient.Login(cfg.Username, cfg.Password); err != nil {
		return false, "", fmt.Errorf("imap login: %w", err)
	}

	if _, err := imapClient.Select(mailbox, true); err != nil {
		return false, "", fmt.Errorf("imap select mailbox: %w", err)
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Now().Add(-48 * time.Hour)
	ids, err := imapClient.Search(criteria)
	if err != nil {
		return false, "", fmt.Errorf("imap search: %w", err)
	}
	if len(ids) == 0 {
		return false, "", nil
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

		if !containsBounceSignature(text) {
			continue
		}

		if token != "" && strings.Contains(text, strings.ToLower(token)) {
			return true, fmt.Sprintf("Bounce detected for probe token in subject: %s", subject), nil
		}

		if strings.Contains(text, strings.ToLower(targetEmail)) {
			return true, fmt.Sprintf("Bounce detected for recipient in subject: %s", subject), nil
		}
	}

	if fetchErr := <-errCh; fetchErr != nil {
		return false, "", fmt.Errorf("imap fetch: %w", fetchErr)
	}

	return false, "", nil
}

func containsBounceSignature(text string) bool {
	signals := []string{
		"delivery status notification (failure)",
		"undeliverable",
		"mail delivery failed",
		"final-recipient",
		"status: 5.",
		"this is the mail system at host",
	}

	for _, signal := range signals {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return false
}
