package service

import (
	"context"
	"crypto/tls"
	"email-verifier-api/internal/repo"
	"email-verifier-api/internal/store"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"
)

type SMTPProbeSender struct {
	repo            *repo.Repository
	torSocksAddr    string
	rotationCounter uint64 // atomic counter for template rotation
}

func NewSMTPProbeSender(r *repo.Repository, torSocksAddr string) *SMTPProbeSender {
	return &SMTPProbeSender{repo: r, torSocksAddr: torSocksAddr}
}

func (s *SMTPProbeSender) SendProbe(ctx context.Context, targetEmail, token string) (string, error) {
	return s.SendProbeForUser(ctx, targetEmail, token, "")
}

func (s *SMTPProbeSender) SendProbeForUser(ctx context.Context, targetEmail, token, userID string) (string, error) {
	var account *store.SMTPAccount
	var err error

	if userID != "" {
		account, err = s.repo.AcquireSMTPAccountForSendByUser(ctx, userID)
	} else {
		account, err = s.repo.AcquireSMTPAccountForSend(ctx)
	}
	if err != nil {
		return "", err
	}
	if account == nil {
		return "", fmt.Errorf("no active smtp account available or all accounts reached daily limit")
	}

	host := normalizeServerHost(account.Host)
	addr := fmt.Sprintf("%s:%d", host, account.Port)
	if err := validateServerHost("host", host); err != nil {
		return "", fmt.Errorf("invalid smtp account configuration: %w", err)
	}
	if err := validatePort("port", account.Port); err != nil {
		return "", fmt.Errorf("invalid smtp account configuration: %w", err)
	}

	var auth smtp.Auth
	if account.Username != "" {
		auth = smtp.PlainAuth("", account.Username, account.Password, host)
	}

	subject := fmt.Sprintf("Email verification probe %s", token)
	body := fmt.Sprintf("This is an automated verification probe. Token: %s\nRecipient: %s\n", token, targetEmail)

	// Get rotating template - increment counter atomically for round-robin
	rotationIndex := int(atomic.AddUint64(&s.rotationCounter, 1))

	var tmpl *store.EmailTemplate
	if userID != "" {
		tmpl, err = s.repo.GetRotatingEmailTemplate(ctx, userID, rotationIndex)
	} else {
		// For non-user requests, use any available template
		tmpl, err = s.repo.GetActiveEmailTemplate(ctx)
	}
	if err == nil && tmpl != nil {
		subject = renderTemplate(tmpl.SubjectTemplate, token, targetEmail, account.Sender)
		body = renderTemplate(tmpl.BodyTemplate, token, targetEmail, account.Sender)
	}

	message := strings.Join([]string{
		fmt.Sprintf("From: %s", account.Sender),
		fmt.Sprintf("To: %s", targetEmail),
		fmt.Sprintf("Subject: %s", subject),
		fmt.Sprintf("Message-ID: <%s@%s>", token, host),
		fmt.Sprintf("X-Verify-Token: %s", token),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	if err := s.sendViaTor(addr, host, account.Port, auth, account.Sender, targetEmail, []byte(message)); err != nil {
		return "", fmt.Errorf("send smtp probe using account %s: %w", account.Username, err)
	}

	return account.ID, nil
}

func (s *SMTPProbeSender) sendViaTor(addr, host string, port int, auth smtp.Auth, from, to string, rawMessage []byte) error {
	var conn net.Conn
	var err error
	torFailed := false
	originalTorErr := error(nil)

	if s.torSocksAddr != "" {
		const maxTorDialAttempts = 3
		for attempt := 1; attempt <= maxTorDialAttempts; attempt++ {
			dialer, derr := proxy.SOCKS5("tcp", s.torSocksAddr, nil, proxy.Direct)
			if derr != nil {
				return fmt.Errorf("create socks5 dialer: %w", derr)
			}

			conn, err = dialer.Dial("tcp", addr)
			if err == nil {
				break
			}
			torFailed = true
			originalTorErr = err

			if !strings.Contains(strings.ToLower(err.Error()), "general socks server failure") || attempt == maxTorDialAttempts {
				break
			}

			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}

	if conn == nil && s.torSocksAddr != "" {
		// Fall back to direct SMTP dial when Tor exits cannot reach provider SMTP ports.
		conn, err = net.Dial("tcp", addr)
	}

	if conn == nil && s.torSocksAddr == "" {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		if torFailed {
			return fmt.Errorf("dial smtp server failed via tor and direct fallback (tor error: %v, direct error: %w)", originalTorErr, err)
		}
		if s.torSocksAddr != "" && strings.Contains(strings.ToLower(err.Error()), "general socks server failure") {
			return fmt.Errorf("dial smtp server through tor: %w (possible causes: invalid smtp host or smtp egress blocked by Tor exit policy)", err)
		}
		return fmt.Errorf("dial smtp server: %w", err)
	}
	defer conn.Close()

	if port == 465 {
		conn = tls.Client(conn, &tls.Config{ServerName: host})
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}

	if port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := wc.Write(rawMessage); err != nil {
		_ = wc.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("close smtp data writer: %w", err)
	}

	return client.Quit()
}

func renderTemplate(tpl, token, email, sender string) string {
	replacer := strings.NewReplacer(
		"{{token}}", token,
		"{{email}}", email,
		"{{sender}}", sender,
	)
	return replacer.Replace(tpl)
}
