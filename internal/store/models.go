package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONStrings []string

func (s JSONStrings) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	raw, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (s *JSONStrings) Scan(src any) error {
	if src == nil {
		*s = JSONStrings{}
		return nil
	}

	var raw []byte
	switch v := src.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = append([]byte(nil), v...)
	default:
		return fmt.Errorf("unsupported JSONStrings source: %T", src)
	}

	if len(raw) == 0 {
		*s = JSONStrings{}
		return nil
	}

	var decoded []string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*s = JSONStrings(decoded)
	return nil
}

type User struct {
	ID           string `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	Email        string `db:"email" json:"email"`
	PasswordHash string `db:"password_hash" json:"-"`
	APIKey       string `db:"api_key" json:"api_key"`
	WebhookURL   string `db:"webhook_url" json:"webhook_url"`
	IsSuperuser  bool   `db:"is_superuser" json:"is_superuser"`
	Active       bool   `db:"active" json:"active"`
	CreatedAt    int64  `db:"created_at" json:"created_at"`
	UpdatedAt    int64  `db:"updated_at" json:"updated_at"`
}

type UserInput struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	APIKey       string
	WebhookURL   string
	IsSuperuser  bool
	Active       bool
}

type VerificationRecord struct {
	ID                string      `db:"id" json:"id"`
	Email             string      `db:"email" json:"email"`
	Domain            string      `db:"domain" json:"domain"`
	UserID            string      `db:"user_id" json:"user_id"`
	Classification    string      `db:"classification" json:"classification"`
	ConfidenceScore   int         `db:"confidence_score" json:"confidence_score"`
	RiskLevel         string      `db:"risk_level" json:"risk_level"`
	Deterministic     bool        `db:"deterministic" json:"deterministic"`
	State             string      `db:"state" json:"state"`
	ReasonCodes       JSONStrings `db:"reason_codes" json:"reason_codes"`
	ProtocolSummary   string      `db:"protocol_summary" json:"protocol_summary"`
	EnrichmentSummary string      `db:"enrichment_summary" json:"enrichment_summary"`
	ExpiresAt         int64       `db:"expires_at" json:"expires_at"`
	LastVerifiedAt    int64       `db:"last_verified_at" json:"last_verified_at"`
	LastEnrichedAt    int64       `db:"last_enriched_at" json:"last_enriched_at"`
	CreatedAt         int64       `db:"created_at" json:"created_at"`
	UpdatedAt         int64       `db:"updated_at" json:"updated_at"`
}

type VerificationCalloutAttempt struct {
	ID             int64  `db:"id" json:"id"`
	VerificationID string `db:"verification_id" json:"verification_id"`
	SMTPHost       string `db:"smtp_host" json:"smtp_host"`
	SMTPPort       int    `db:"smtp_port" json:"smtp_port"`
	Stage          string `db:"stage" json:"stage"`
	Recipient      string `db:"recipient" json:"recipient"`
	Outcome        string `db:"outcome" json:"outcome"`
	SMTPCode       int    `db:"smtp_code" json:"smtp_code"`
	SMTPMessage    string `db:"smtp_message" json:"smtp_message"`
	DurationMS     int64  `db:"duration_ms" json:"duration_ms"`
	CreatedAt      int64  `db:"created_at" json:"created_at"`
}

type DomainBaseline struct {
	Domain         string `db:"domain" json:"domain"`
	MXFingerprint  string `db:"mx_fingerprint" json:"mx_fingerprint"`
	Classification string `db:"classification" json:"classification"`
	SampleAddress  string `db:"sample_address" json:"sample_address"`
	SMTPHost       string `db:"smtp_host" json:"smtp_host"`
	SMTPCode       int    `db:"smtp_code" json:"smtp_code"`
	SMTPMessage    string `db:"smtp_message" json:"smtp_message"`
	CheckedAt      int64  `db:"checked_at" json:"checked_at"`
	ExpiresAt      int64  `db:"expires_at" json:"expires_at"`
	CreatedAt      int64  `db:"created_at" json:"created_at"`
	UpdatedAt      int64  `db:"updated_at" json:"updated_at"`
}

type EnrichmentEvidence struct {
	ID             string `db:"id" json:"id"`
	VerificationID string `db:"verification_id" json:"verification_id"`
	Source         string `db:"source" json:"source"`
	Kind           string `db:"kind" json:"kind"`
	Signal         string `db:"signal" json:"signal"`
	Weight         int    `db:"weight" json:"weight"`
	Summary        string `db:"summary" json:"summary"`
	CreatedAt      int64  `db:"created_at" json:"created_at"`
}
