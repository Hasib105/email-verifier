package service

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var hostnamePattern = regexp.MustCompile(`(?i)^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)
var commonAtHostPattern = regexp.MustCompile(`(?i)^(smtp|imap)@([a-z0-9.-]+)$`)

func normalizeServerHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if matches := commonAtHostPattern.FindStringSubmatch(host); len(matches) == 3 {
		return matches[1] + "." + matches[2]
	}
	return host
}

func inferIMAPHost(smtpHost string) string {
	smtpHost = normalizeServerHost(smtpHost)
	if strings.HasPrefix(smtpHost, "smtp.") {
		return "imap." + strings.TrimPrefix(smtpHost, "smtp.")
	}
	return smtpHost
}

func validateServerHost(fieldName, host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("%s is required", fieldName)
	}

	// Prevent common misconfiguration where an email is entered as host (e.g. smtp@gmail.com).
	if strings.Contains(host, "@") || strings.Contains(host, "://") || strings.ContainsAny(host, " /\\") {
		return fmt.Errorf("%s must be a hostname (example: smtp.gmail.com), got %q", fieldName, host)
	}

	if net.ParseIP(host) != nil {
		return nil
	}

	if !hostnamePattern.MatchString(host) {
		return fmt.Errorf("%s must be a valid hostname or IP, got %q", fieldName, host)
	}

	return nil
}

func validatePort(fieldName string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", fieldName)
	}
	return nil
}
