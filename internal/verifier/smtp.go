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
	"sync"
	"time"
)

type VerifyResult struct {
	Status               string `json:"status"`
	Message              string `json:"message"`
	Email                string `json:"email"`
	RequireProbe         bool   `json:"require_probe"`
	Deterministic        bool   `json:"deterministic"`
	ReasonCode           string `json:"reason_code"`
	SignalSummary        string `json:"signal_summary"`
	DirectAvailability   string `json:"direct_availability"`
	UsedRoutingFallback  bool   `json:"used_routing_fallback"`
	UsedStrictProviderMX bool   `json:"used_strict_provider_mx"`
}

type HealthSnapshot struct {
	DirectSMTPStatus string `json:"direct_smtp_status"`
	LastCheckedAt    int64  `json:"last_checked_at"`
	Message          string `json:"message"`
}

type EmailVerifier struct {
	FromEmail  string
	EHLODomain string
	Timeout    time.Duration
	Semaphore  chan struct{}

	healthMu          sync.RWMutex
	directSMTPStatus  string
	lastCheckedAt     int64
	lastHealthMessage string
}

type mailRouting struct {
	domain        string
	hosts         []string
	usedAFallback bool
}

type calloutAttempt struct {
	host       string
	stage      string
	connected  bool
	outcome    string
	code       int
	message    string
	durationMS int64
}

type recipientCheckResult struct {
	outcome      string
	code         int
	message      string
	attempts     []calloutAttempt
	availability string
}

func New(fromEmail, ehloDomain string, maxConcurrency int, timeout time.Duration) *EmailVerifier {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}

	return &EmailVerifier{
		FromEmail:         fromEmail,
		EHLODomain:        ehloDomain,
		Timeout:           timeout,
		Semaphore:         make(chan struct{}, maxConcurrency),
		directSMTPStatus:  "unknown",
		lastHealthMessage: "no direct SMTP checks have completed yet",
	}
}

func (v *EmailVerifier) HealthSnapshot() HealthSnapshot {
	v.healthMu.RLock()
	defer v.healthMu.RUnlock()

	return HealthSnapshot{
		DirectSMTPStatus: v.directSMTPStatus,
		LastCheckedAt:    v.lastCheckedAt,
		Message:          v.lastHealthMessage,
	}
}

func (v *EmailVerifier) Verify(rawEmail string) VerifyResult {
	v.Semaphore <- struct{}{}
	defer func() { <-v.Semaphore }()

	email, domain, err := normalizeEmail(rawEmail)
	if err != nil {
		return VerifyResult{
			Status:             "invalid",
			Message:            err.Error(),
			Email:              strings.ToLower(strings.TrimSpace(rawEmail)),
			Deterministic:      true,
			ReasonCode:         "syntax_invalid",
			SignalSummary:      "Address failed syntax validation before any network checks.",
			DirectAvailability: "unknown",
		}
	}

	result := VerifyResult{Email: email, DirectAvailability: "unknown"}

	if isDisposableDomain(domain) {
		result.Status = "disposable"
		result.Message = "disposable domain detected"
		result.Deterministic = true
		result.ReasonCode = "disposable_domain"
		result.SignalSummary = "Domain is on the local disposable mailbox list."
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.Timeout)
	defer cancel()

	routing, routingErr := resolveMailRouting(ctx, domain)
	if routingErr != nil {
		if errors.Is(routingErr, errNoMailRouting) {
			result.Status = "invalid"
			result.Message = "no MX or A/AAAA records"
			result.Deterministic = true
			result.ReasonCode = "no_mail_routing"
			result.SignalSummary = "Domain has no MX records and no A/AAAA fallback for mail routing."
			return result
		}

		result.Status = "error"
		result.Message = routingErr.Error()
		result.RequireProbe = true
		result.Deterministic = false
		result.ReasonCode = "direct_path_unavailable"
		result.SignalSummary = "Direct SMTP routing could not be established; probe fallback is required."
		v.setHealth("degraded", result.Message)
		result.DirectAvailability = "degraded"
		return result
	}

	strictProvider := routingUsesStrictProvider(routing)
	result.UsedRoutingFallback = routing.usedAFallback
	result.UsedStrictProviderMX = strictProvider

	check := v.checkRecipient(ctx, routing, email)
	result.DirectAvailability = check.availability
	if check.availability != "unknown" {
		v.setHealth(check.availability, check.message)
	}

	switch check.outcome {
	case "rejected":
		result.Status = "invalid"
		result.Message = check.message
		result.Deterministic = true
		result.ReasonCode = "hard_rcpt_reject"
		result.SignalSummary = directSignalSummary("Recipient MX explicitly rejected RCPT.", strictProvider, routing.usedAFallback)
	case "accepted":
		if strictProvider {
			result.Status = "unknown"
			result.Message = fmt.Sprintf("%s; provider acceptance is not treated as mailbox proof", check.message)
			result.RequireProbe = true
			result.Deterministic = false
			result.ReasonCode = "strict_provider_inconclusive"
			result.SignalSummary = directSignalSummary("Recipient MX accepted RCPT, but the provider is known to make acceptance a weak mailbox-existence signal.", true, routing.usedAFallback)
			return result
		}

		result.Status = "valid"
		result.Message = check.message
		result.Deterministic = false
		result.ReasonCode = "direct_accept_non_strict"
		result.SignalSummary = directSignalSummary("Recipient MX accepted RCPT on a non-strict provider.", false, routing.usedAFallback)
	case "tempfail":
		result.Status = "greylisted"
		result.Message = check.message
		result.RequireProbe = true
		result.Deterministic = false
		result.ReasonCode = "temporary_failure"
		result.SignalSummary = directSignalSummary("Recipient MX returned a temporary failure; probe fallback is required.", strictProvider, routing.usedAFallback)
	case "policy":
		result.Status = "unknown"
		result.Message = check.message
		result.RequireProbe = true
		result.Deterministic = false
		result.ReasonCode = "provider_policy_block"
		result.SignalSummary = directSignalSummary("Recipient MX blocked mailbox verification or enforced a policy restriction.", strictProvider, routing.usedAFallback)
	default:
		result.Status = "error"
		result.Message = check.message
		result.RequireProbe = true
		result.Deterministic = false
		result.ReasonCode = "direct_path_unavailable"
		result.SignalSummary = directSignalSummary("Direct SMTP callouts were unavailable or inconclusive; probe fallback is required.", strictProvider, routing.usedAFallback)
	}

	return result
}

var errNoMailRouting = errors.New("no mail routing")

func resolveMailRouting(ctx context.Context, domain string) (mailRouting, error) {
	mxs, err := net.DefaultResolver.LookupMX(ctx, domain)
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
			return mailRouting{domain: domain, hosts: hosts}, nil
		}
	}

	if _, ipErr := net.DefaultResolver.LookupIPAddr(ctx, domain); ipErr == nil {
		return mailRouting{
			domain:        domain,
			hosts:         []string{domain},
			usedAFallback: true,
		}, nil
	}

	if err != nil {
		return mailRouting{}, fmt.Errorf("resolve mail routing: %w", err)
	}
	return mailRouting{}, errNoMailRouting
}

func (v *EmailVerifier) checkRecipient(ctx context.Context, routing mailRouting, recipient string) recipientCheckResult {
	result := recipientCheckResult{
		outcome:      "error",
		message:      "no SMTP hosts attempted",
		availability: "unknown",
	}

	connected := false
	for _, host := range routing.hosts {
		attempt := v.callHost(ctx, host, recipient)
		result.attempts = append(result.attempts, attempt)
		if attempt.connected {
			connected = true
			result.availability = "available"
		}

		switch attempt.outcome {
		case "accepted":
			return recipientCheckResult{
				outcome:      "accepted",
				code:         attempt.code,
				message:      attempt.message,
				attempts:     result.attempts,
				availability: availabilityStatus(connected, true),
			}
		case "rejected":
			return recipientCheckResult{
				outcome:      "rejected",
				code:         attempt.code,
				message:      attempt.message,
				attempts:     result.attempts,
				availability: availabilityStatus(connected, true),
			}
		case "policy":
			result.outcome = "policy"
			result.code = attempt.code
			result.message = attempt.message
		case "tempfail":
			if result.outcome == "error" {
				result.outcome = "tempfail"
				result.code = attempt.code
				result.message = attempt.message
			}
		default:
			if result.message == "no SMTP hosts attempted" {
				result.message = attempt.message
			}
		}
	}

	result.availability = availabilityStatus(connected, len(result.attempts) > 0)
	return result
}

func (v *EmailVerifier) callHost(ctx context.Context, host, recipient string) calloutAttempt {
	start := time.Now()
	attempt := calloutAttempt{
		host:    host,
		stage:   "connect",
		outcome: "error",
	}

	dialer := &net.Dialer{Timeout: v.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, "25"))
	if err != nil {
		attempt.message = err.Error()
		attempt.durationMS = time.Since(start).Milliseconds()
		return attempt
	}
	attempt.connected = true
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		attempt.message = err.Error()
		attempt.durationMS = time.Since(start).Milliseconds()
		return attempt
	}
	defer client.Close()

	attempt.stage = "ehlo"
	if err := client.Hello(v.EHLODomain); err != nil {
		attempt.outcome, attempt.code, attempt.message = classifySMTPErr(err, "EHLO failed")
		attempt.durationMS = time.Since(start).Milliseconds()
		return attempt
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		_ = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		_ = client.Hello(v.EHLODomain)
	}

	attempt.stage = "mail_from"
	if err := client.Mail(v.FromEmail); err != nil {
		attempt.outcome, attempt.code, attempt.message = classifySMTPErr(err, "MAIL FROM rejected")
		attempt.durationMS = time.Since(start).Milliseconds()
		return attempt
	}

	attempt.stage = "rcpt_to"
	if err := client.Rcpt(recipient); err != nil {
		attempt.outcome, attempt.code, attempt.message = classifySMTPErr(err, "RCPT TO rejected")
		attempt.durationMS = time.Since(start).Milliseconds()
		return attempt
	}

	attempt.outcome = "accepted"
	attempt.code = 250
	attempt.message = "250 recipient accepted"
	attempt.durationMS = time.Since(start).Milliseconds()
	return attempt
}

func classifySMTPErr(err error, fallback string) (string, int, string) {
	if smtpErr, ok := err.(*textproto.Error); ok {
		code := smtpErr.Code
		text := strings.TrimSpace(smtpErr.Msg)
		lower := strings.ToLower(text)
		switch {
		case code >= 500 && code <= 559:
			if strings.Contains(text, "5.7") || strings.Contains(lower, "policy") || strings.Contains(lower, "verification disabled") || strings.Contains(lower, "access denied") || strings.Contains(lower, "cannot verify") {
				return "policy", code, fmt.Sprintf("%d %s", code, text)
			}
			return "rejected", code, fmt.Sprintf("%d %s", code, text)
		case code >= 400 && code <= 499:
			return "tempfail", code, fmt.Sprintf("%d %s", code, text)
		default:
			return "error", code, fmt.Sprintf("%d %s", code, text)
		}
	}

	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "timeout") {
		return "tempfail", 0, msg
	}

	return "error", 0, fmt.Sprintf("%s: %s", fallback, msg)
}

func normalizeEmail(raw string) (string, string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", "", errors.New("invalid syntax")
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

func routingUsesStrictProvider(routing mailRouting) bool {
	for _, host := range routing.hosts {
		if isStrictProvider(host) {
			return true
		}
	}
	return false
}

func isStrictProvider(mx string) bool {
	patterns := []string{
		"google.com",
		"googlemail.com",
		"gmail-smtp-in.l.google.com",
		"outlook.com",
		"hotmail.com",
		"microsoft.com",
		"protection.outlook.com",
		"mail.protection.outlook.com",
		"yahoodns.net",
		"yahoo.com",
		"icloud.com",
		"me.com",
		"apple.com",
	}

	lower := strings.ToLower(mx)
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func directSignalSummary(base string, strictProvider, usedAFallback bool) string {
	parts := []string{base}
	if strictProvider {
		parts = append(parts, "Provider is on the local strict-provider list.")
	}
	if usedAFallback {
		parts = append(parts, "Domain used A/AAAA fallback because no MX records were present.")
	}
	return strings.Join(parts, " ")
}

func (v *EmailVerifier) setHealth(status, message string) {
	if status == "" {
		status = "unknown"
	}

	v.healthMu.Lock()
	defer v.healthMu.Unlock()
	v.directSMTPStatus = status
	v.lastCheckedAt = time.Now().Unix()
	v.lastHealthMessage = message
}

func availabilityStatus(connected, attempted bool) string {
	switch {
	case connected:
		return "available"
	case attempted:
		return "degraded"
	default:
		return "unknown"
	}
}

func isDisposableDomain(domain string) bool {
	disposables := map[string]bool{
		"10minutemail.com":  true,
		"fakeinbox.com":     true,
		"getnada.com":       true,
		"guerrillamail.com": true,
		"mailinator.com":    true,
		"temp-mail.org":     true,
		"tempmail.com":      true,
		"throwaway.email":   true,
		"yopmail.com":       true,
	}
	return disposables[strings.ToLower(domain)]
}
