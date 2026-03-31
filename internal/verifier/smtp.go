package verifier

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type VerifyResult struct {
	Status       string `json:"status"`        // valid, invalid, unknown, greylisted, disposable, error
	Message      string `json:"message"`
	Email        string `json:"email"`
	RequireProbe bool   `json:"require_probe"` // true if EHLO check was denied and needs probe fallback
}

type EmailVerifier struct {
	FromEmail  string
	EHLODomain string
	Timeout    time.Duration
	ProxyAddr  string
	Semaphore  chan struct{}
}

func New(fromEmail, ehloDomain, proxyAddr string, maxConcurrency int, timeout time.Duration) *EmailVerifier {
	return &EmailVerifier{
		FromEmail:  fromEmail,
		EHLODomain: ehloDomain,
		ProxyAddr:  proxyAddr,
		Timeout:    timeout,
		Semaphore:  make(chan struct{}, maxConcurrency),
	}
}

// isStrictProvider returns true if the domain is known to reject EHLO-based verification
func isStrictProvider(mx string) bool {
	strictPatterns := []string{
		"google.com",
		"googlemail.com",
		"gmail-smtp-in.l.google.com",
		"outlook.com",
		"hotmail.com",
		"microsoft.com",
		"protection.outlook.com",
		"mail.protection.outlook.com",
		"yahoo.com",
		"yahoodns.net",
		"icloud.com",
		"me.com",
		"apple.com",
	}
	mxLower := strings.ToLower(mx)
	for _, pattern := range strictPatterns {
		if strings.Contains(mxLower, pattern) {
			return true
		}
	}
	return false
}

// isEHLODeniedError checks if the error message indicates EHLO-based verification was blocked
func isEHLODeniedError(msg string) bool {
	deniedPatterns := []string{
		"cannot verify user",
		"try again later",
		"rate limit",
		"too many",
		"temporarily deferred",
		"service unavailable",
		"authentication required",
		"relay access denied",
		"sender verify failed",
		"550 5.7",
		"421 4.7",
		"450 4.2",
		"451 4.7",
	}
	msgLower := strings.ToLower(msg)
	for _, pattern := range deniedPatterns {
		if strings.Contains(msgLower, pattern) {
			return true
		}
	}
	return false
}

func (v *EmailVerifier) Verify(email string) VerifyResult {
	// Acquire semaphore (limit concurrent Tor connections)
	v.Semaphore <- struct{}{}
	defer func() { <-v.Semaphore }()

	email = strings.ToLower(strings.TrimSpace(email))
	res := VerifyResult{Email: email}

	if !isValidEmailSyntax(email) {
		res.Status = "invalid"
		res.Message = "invalid syntax"
		return res
	}

	if isDisposableDomain(strings.Split(email, "@")[1]) {
		res.Status = "disposable"
		res.Message = "disposable domain detected"
		return res
	}

	domain := strings.Split(email, "@")[1]
	mxs, err := net.LookupMX(domain)
	if err != nil || len(mxs) == 0 {
		res.Status = "invalid"
		res.Message = "no MX records"
		return res
	}

	mx := strings.TrimSuffix(mxs[0].Host, ".")

	// Check if this is a strict provider that blocks EHLO verification
	if isStrictProvider(mx) {
		res.Status = "unknown"
		res.Message = fmt.Sprintf("strict provider detected (%s); requires probe-based verification", mx)
		res.RequireProbe = true
		return res
	}

	conn, err := v.dialSMTP(mx)
	if err != nil {
		res.Status = "error"
		res.Message = fmt.Sprintf("connection failed: %v", err)
		res.RequireProbe = true
		return res
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, mx)
	if err != nil {
		res.Status = "error"
		res.Message = err.Error()
		res.RequireProbe = true
		return res
	}
	defer client.Close()

	if err := client.Hello(v.EHLODomain); err != nil {
		res.Status = "error"
		res.Message = "EHLO failed"
		res.RequireProbe = true
		return res
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: mx, InsecureSkipVerify: false}
		_ = client.StartTLS(tlsConfig) // Ignore error, continue if fails
		client.Hello(v.EHLODomain)
	}

	if err := client.Mail(v.FromEmail); err != nil {
		errMsg := err.Error()
		if isEHLODeniedError(errMsg) {
			res.Status = "unknown"
			res.Message = fmt.Sprintf("MAIL FROM rejected; probe required: %s", errMsg)
			res.RequireProbe = true
		} else {
			res.Status = "error"
			res.Message = fmt.Sprintf("MAIL FROM rejected: %s", errMsg)
			res.RequireProbe = true
		}
		return res
	}

	rcptErr := client.Rcpt(email)
	if rcptErr == nil {
		res.Status = "valid"
		res.Message = "250 Accepted"
		return res
	}

	// Parse Error Codes
	if smtpErr, ok := rcptErr.(*textproto.Error); ok {
		errMsg := fmt.Sprintf("%d %s", smtpErr.Code, smtpErr.Msg)

		// Check if this is an EHLO denial that needs probe fallback
		if isEHLODeniedError(smtpErr.Msg) {
			res.Status = "unknown"
			res.Message = errMsg
			res.RequireProbe = true
			return res
		}

		if smtpErr.Code >= 550 && smtpErr.Code <= 559 {
			res.Status = "invalid"
			res.Message = errMsg
		} else if smtpErr.Code == 450 || smtpErr.Code == 451 || smtpErr.Code == 452 {
			res.Status = "greylisted"
			res.Message = errMsg
			res.RequireProbe = true
		} else if smtpErr.Code >= 400 && smtpErr.Code < 500 {
			res.Status = "unknown"
			res.Message = errMsg
			res.RequireProbe = true
		} else {
			res.Status = "unknown"
			res.Message = errMsg
			res.RequireProbe = true
		}
	} else {
		res.Status = "unknown"
		res.Message = rcptErr.Error()
		res.RequireProbe = true
	}

	return res
}

func (v *EmailVerifier) dialSMTP(host string) (net.Conn, error) {
	addr := net.JoinHostPort(host, "25")

	// If no proxy configured, use direct (not recommended for privacy)
	if v.ProxyAddr == "" {
		d := &net.Dialer{Timeout: v.Timeout}
		return d.Dial("tcp", addr)
	}

	// SOCKS5 Dialer (Tor) - All connections go through Tor for anonymity
	dialer, err := proxy.SOCKS5("tcp", v.ProxyAddr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return dialer.Dial("tcp", addr)
}

func isValidEmailSyntax(email string) bool {
	const pattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}

func isDisposableDomain(domain string) bool {
	// Expand this list in production
	disposables := map[string]bool{
		"mailinator.com": true, "yopmail.com": true, "10minutemail.com": true,
		"guerrillamail.com": true, "tempmail.com": true, "throwaway.email": true,
		"temp-mail.org": true, "fakeinbox.com": true, "getnada.com": true,
	}
	return disposables[domain]
}