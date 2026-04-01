package verifier

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"sort"
	"strings"
	"time"
)

type MailRouting struct {
	Domain        string
	Hosts         []string
	Fingerprint   string
	UsedAFallback bool
}

type CalloutAttempt struct {
	Host       string
	Port       int
	Stage      string
	Recipient  string
	Outcome    string
	Code       int
	Message    string
	DurationMS int64
}

type RecipientCheckResult struct {
	Outcome  string
	Code     int
	Message  string
	Attempts []CalloutAttempt
}

type EmailVerifier struct {
	MailFrom      string
	EHLODomain    string
	Timeout       time.Duration
	Semaphore     chan struct{}
	Resolver      DNSResolver
	CalloutEngine CalloutEngine
}

type DNSResolver interface {
	LookupMX(ctx context.Context, domain string) ([]*net.MX, error)
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type CalloutEngine interface {
	CheckRecipient(ctx context.Context, routing MailRouting, mailFrom, ehloDomain, recipient string) RecipientCheckResult
}

type NetResolver struct{}

func (NetResolver) LookupMX(ctx context.Context, domain string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(ctx, domain)
}

func (NetResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

type SMTPDialer struct {
	Timeout time.Duration
}

func New(mailFrom, ehloDomain string, maxConcurrency int, timeout time.Duration) *EmailVerifier {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}
	return &EmailVerifier{
		MailFrom:      mailFrom,
		EHLODomain:    ehloDomain,
		Timeout:       timeout,
		Semaphore:     make(chan struct{}, maxConcurrency),
		Resolver:      NetResolver{},
		CalloutEngine: SMTPDialer{Timeout: timeout},
	}
}

func NormalizeEmail(raw string) (string, string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", "", errors.New("email is required")
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(parsed.Address, email) {
		return "", "", errors.New("invalid syntax")
	}

	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" {
		return "", "", errors.New("invalid syntax")
	}
	if len(local) > 64 || len(domain) > 253 {
		return "", "", errors.New("invalid syntax")
	}
	if strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return "", "", errors.New("invalid syntax")
	}
	if strings.ContainsAny(domain, " /\\") {
		return "", "", errors.New("invalid syntax")
	}

	return email, domain, nil
}

func (v *EmailVerifier) Resolve(ctx context.Context, domain string) (MailRouting, error) {
	v.Semaphore <- struct{}{}
	defer func() { <-v.Semaphore }()

	mxs, err := v.Resolver.LookupMX(ctx, domain)
	if err == nil && len(mxs) > 0 {
		sort.Slice(mxs, func(i, j int) bool {
			if mxs[i].Pref == mxs[j].Pref {
				return mxs[i].Host < mxs[j].Host
			}
			return mxs[i].Pref < mxs[j].Pref
		})

		hosts := make([]string, 0, len(mxs))
		for _, mx := range mxs {
			host := strings.TrimSuffix(strings.ToLower(mx.Host), ".")
			if host != "" {
				hosts = append(hosts, host)
			}
		}
		if len(hosts) > 0 {
			return MailRouting{
				Domain:      domain,
				Hosts:       hosts,
				Fingerprint: strings.Join(hosts, ","),
			}, nil
		}
	}

	if _, ipErr := v.Resolver.LookupIPAddr(ctx, domain); ipErr == nil {
		return MailRouting{
			Domain:        domain,
			Hosts:         []string{domain},
			Fingerprint:   domain,
			UsedAFallback: true,
		}, nil
	}

	if err != nil {
		return MailRouting{}, fmt.Errorf("resolve mail routing: %w", err)
	}
	return MailRouting{}, fmt.Errorf("resolve mail routing: no MX or A/AAAA records")
}

func (v *EmailVerifier) CheckRecipient(ctx context.Context, routing MailRouting, recipient string) RecipientCheckResult {
	v.Semaphore <- struct{}{}
	defer func() { <-v.Semaphore }()
	return v.CalloutEngine.CheckRecipient(ctx, routing, v.MailFrom, v.EHLODomain, recipient)
}

func (d SMTPDialer) CheckRecipient(ctx context.Context, routing MailRouting, mailFrom, ehloDomain, recipient string) RecipientCheckResult {
	result := RecipientCheckResult{
		Outcome: "error",
		Message: "no SMTP hosts attempted",
	}

	for _, host := range routing.Hosts {
		attempt := d.callHost(ctx, host, mailFrom, ehloDomain, recipient)
		result.Attempts = append(result.Attempts, attempt)

		switch attempt.Outcome {
		case "accepted":
			return RecipientCheckResult{
				Outcome:  "accepted",
				Code:     attempt.Code,
				Message:  attempt.Message,
				Attempts: result.Attempts,
			}
		case "rejected":
			return RecipientCheckResult{
				Outcome:  "rejected",
				Code:     attempt.Code,
				Message:  attempt.Message,
				Attempts: result.Attempts,
			}
		case "policy":
			result.Outcome = "policy"
			result.Code = attempt.Code
			result.Message = attempt.Message
		case "tempfail":
			if result.Outcome == "error" {
				result.Outcome = "tempfail"
				result.Code = attempt.Code
				result.Message = attempt.Message
			}
		default:
			if result.Message == "no SMTP hosts attempted" {
				result.Message = attempt.Message
			}
		}
	}

	return result
}

func (d SMTPDialer) callHost(ctx context.Context, host, mailFrom, ehloDomain, recipient string) CalloutAttempt {
	start := time.Now()
	attempt := CalloutAttempt{
		Host:      host,
		Port:      25,
		Stage:     "connect",
		Recipient: recipient,
		Outcome:   "error",
	}

	dialer := &net.Dialer{Timeout: d.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, "25"))
	if err != nil {
		attempt.Message = err.Error()
		attempt.DurationMS = time.Since(start).Milliseconds()
		return attempt
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		attempt.Message = err.Error()
		attempt.DurationMS = time.Since(start).Milliseconds()
		return attempt
	}
	defer client.Close()

	attempt.Stage = "ehlo"
	if err := client.Hello(ehloDomain); err != nil {
		attempt.Outcome, attempt.Code, attempt.Message = classifySMTPErr(err, "EHLO failed")
		attempt.DurationMS = time.Since(start).Milliseconds()
		return attempt
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		_ = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		_ = client.Hello(ehloDomain)
	}

	attempt.Stage = "mail_from"
	if err := client.Mail(mailFrom); err != nil {
		attempt.Outcome, attempt.Code, attempt.Message = classifySMTPErr(err, "MAIL FROM rejected")
		attempt.DurationMS = time.Since(start).Milliseconds()
		return attempt
	}

	attempt.Stage = "rcpt_to"
	if err := client.Rcpt(recipient); err != nil {
		attempt.Outcome, attempt.Code, attempt.Message = classifySMTPErr(err, "RCPT TO rejected")
		attempt.DurationMS = time.Since(start).Milliseconds()
		return attempt
	}

	attempt.Outcome = "accepted"
	attempt.Code = 250
	attempt.Message = "recipient accepted"
	attempt.DurationMS = time.Since(start).Milliseconds()
	return attempt
}

func classifySMTPErr(err error, fallback string) (string, int, string) {
	if err == nil {
		return "accepted", 250, "recipient accepted"
	}

	msg := err.Error()
	if smtpErr, ok := err.(*textproto.Error); ok {
		code := smtpErr.Code
		text := strings.TrimSpace(smtpErr.Msg)
		switch {
		case code >= 500 && code <= 559:
			if strings.Contains(text, "5.7") || strings.Contains(strings.ToLower(text), "policy") || strings.Contains(strings.ToLower(text), "access denied") {
				return "policy", code, fmt.Sprintf("%d %s", code, text)
			}
			return "rejected", code, fmt.Sprintf("%d %s", code, text)
		case code >= 400 && code <= 499:
			return "tempfail", code, fmt.Sprintf("%d %s", code, text)
		default:
			return "error", code, fmt.Sprintf("%d %s", code, text)
		}
	}

	if strings.Contains(strings.ToLower(msg), "timeout") {
		return "tempfail", 0, msg
	}

	return "error", 0, fmt.Sprintf("%s: %s", fallback, msg)
}
