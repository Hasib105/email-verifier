package serviceutil

import "strings"

func ContainsBounceSignature(text string) bool {
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

func SummarizeLocalSignals(email, base string) string {
	local, domain, _ := strings.Cut(strings.ToLower(email), "@")
	notes := []string{}

	if isConsumerMailboxDomain(domain) {
		notes = append(notes, "Consumer mailbox domain.")
	}
	if isRoleMailbox(local) {
		notes = append(notes, "Role-based local part.")
	}

	if len(notes) == 0 {
		return base
	}
	if base == "" {
		return strings.Join(notes, " ")
	}
	return base + " " + strings.Join(notes, " ")
}

func ProbePathForReason(reasonCode string) string {
	switch reasonCode {
	case "direct_path_unavailable":
		return "probe_bounce"
	default:
		return "hybrid"
	}
}

func BounceReasonCode(matchKind string) string {
	switch matchKind {
	case "token_match":
		return "bounce_token_match"
	case "recipient_match":
		return "bounce_recipient_match"
	default:
		return "bounce_generic_dsn"
	}
}

func BounceSignalSummary(matchKind string) string {
	switch matchKind {
	case "token_match":
		return "Bounce evidence matched the unique probe token."
	case "recipient_match":
		return "Bounce evidence matched the recipient address."
	default:
		return "Bounce evidence matched a generic delivery-status notification."
	}
}

func isConsumerMailboxDomain(domain string) bool {
	switch strings.ToLower(domain) {
	case "gmail.com", "googlemail.com", "hotmail.com", "icloud.com", "me.com", "outlook.com", "yahoo.com":
		return true
	default:
		return false
	}
}

func isRoleMailbox(local string) bool {
	switch strings.ToLower(local) {
	case "admin", "billing", "contact", "hello", "info", "legal", "sales", "support", "team":
		return true
	default:
		return false
	}
}
