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
	Status  string `json:"status"`  // valid, invalid, unknown, greylisted, disposable, error
	Message string `json:"message"`
	Email   string `json:"email"`
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
	conn, err := v.dialSMTP(mx)
	if err != nil {
		res.Status = "error"
		res.Message = fmt.Sprintf("connection failed: %v", err)
		return res
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, mx)
	if err != nil {
		res.Status = "error"
		res.Message = err.Error()
		return res
	}
	defer client.Close()

	if err := client.Hello(v.EHLODomain); err != nil {
		res.Status = "error"
		res.Message = "EHLO failed"
		return res
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: mx, InsecureSkipVerify: false}
		_ = client.StartTLS(tlsConfig) // Ignore error, continue if fails
		client.Hello(v.EHLODomain)
	}

	if err := client.Mail(v.FromEmail); err != nil {
		res.Status = "error"
		res.Message = "MAIL FROM rejected"
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
		if smtpErr.Code >= 550 && smtpErr.Code <= 559 {
			res.Status = "invalid"
			res.Message = fmt.Sprintf("%d %s", smtpErr.Code, smtpErr.Msg)
		} else if smtpErr.Code == 450 || smtpErr.Code == 451 {
			res.Status = "greylisted"
			res.Message = fmt.Sprintf("%d %s", smtpErr.Code, smtpErr.Msg)
		} else {
			res.Status = "unknown"
			res.Message = fmt.Sprintf("%d %s", smtpErr.Code, smtpErr.Msg)
		}
	} else {
		res.Status = "unknown"
		res.Message = rcptErr.Error()
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

	// SOCKS5 Dialer (Tor)
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
	}
	return disposables[domain]
}