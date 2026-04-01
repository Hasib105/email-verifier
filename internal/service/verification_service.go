package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"email-verifier-api/internal/repo"
	"email-verifier-api/internal/store"
	"email-verifier-api/internal/verifier"

	"github.com/google/uuid"
)

type ServiceConfig struct {
	DeliverableTTL    time.Duration
	UndeliverableTTL  time.Duration
	AcceptAllTTL      time.Duration
	UnknownTTL        time.Duration
	DomainBaselineTTL time.Duration
	EnrichmentWorkers int
}

var (
	ErrVerificationNotFound  = errors.New("verification not found")
	ErrVerificationForbidden = errors.New("verification access denied")
)

type VerifyResponse struct {
	ID                string                             `json:"id"`
	Email             string                             `json:"email"`
	Domain            string                             `json:"domain"`
	Classification    string                             `json:"classification"`
	ConfidenceScore   int                                `json:"confidence_score"`
	RiskLevel         string                             `json:"risk_level"`
	Deterministic     bool                               `json:"deterministic"`
	State             string                             `json:"state"`
	ReasonCodes       []string                           `json:"reason_codes"`
	ProtocolSummary   string                             `json:"protocol_summary"`
	EnrichmentSummary string                             `json:"enrichment_summary"`
	ExpiresAt         int64                              `json:"expires_at"`
	LastVerifiedAt    int64                              `json:"last_verified_at"`
	LastEnrichedAt    int64                              `json:"last_enriched_at"`
	Cached            bool                               `json:"cached"`
	Evidence          []store.EnrichmentEvidence         `json:"evidence,omitempty"`
	Callouts          []store.VerificationCalloutAttempt `json:"callouts,omitempty"`
}

type EmailVerificationService struct {
	verifier        *verifier.EmailVerifier
	repo            *repo.Repository
	enricher        *EnrichmentService
	cfg             ServiceConfig
	enrichmentQueue chan string
}

func NewEmailVerificationService(v *verifier.EmailVerifier, r *repo.Repository, enricher *EnrichmentService, cfg ServiceConfig) *EmailVerificationService {
	if cfg.EnrichmentWorkers <= 0 {
		cfg.EnrichmentWorkers = 1
	}
	return &EmailVerificationService{
		verifier:        v,
		repo:            r,
		enricher:        enricher,
		cfg:             cfg,
		enrichmentQueue: make(chan string, 256),
	}
}

func (s *EmailVerificationService) StartBackground(ctx context.Context) {
	workers := s.cfg.EnrichmentWorkers
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		go s.runEnrichmentWorker(ctx)
	}
}

func (s *EmailVerificationService) runEnrichmentWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case verificationID := <-s.enrichmentQueue:
			if err := s.enrichVerification(ctx, verificationID); err != nil {
				log.Printf("warning: enrichment failed for %s: %v", verificationID, err)
			}
		}
	}
}

func (s *EmailVerificationService) VerifyEmail(ctx context.Context, email string, user *store.User) (VerifyResponse, error) {
	return s.verifyEmailWithBaselineCache(ctx, email, user, map[string]*store.DomainBaseline{})
}

func (s *EmailVerificationService) VerifyEmailBatch(ctx context.Context, emails []string, user *store.User) ([]VerifyResponse, int) {
	cache := map[string]*store.DomainBaseline{}
	items := make([]VerifyResponse, 0, len(emails))
	accepted := 0

	for _, raw := range emails {
		resp, err := s.verifyEmailWithBaselineCache(ctx, raw, user, cache)
		if err != nil {
			items = append(items, VerifyResponse{
				Email:           strings.ToLower(strings.TrimSpace(raw)),
				Classification:  "unknown",
				ConfidenceScore: 0,
				RiskLevel:       "high",
				Deterministic:   false,
				State:           "completed",
				ReasonCodes:     []string{"verification_error"},
				ProtocolSummary: err.Error(),
				Cached:          false,
			})
			continue
		}
		items = append(items, resp)
		accepted++
	}

	return items, accepted
}

func (s *EmailVerificationService) verifyEmailWithBaselineCache(ctx context.Context, rawEmail string, user *store.User, baselineCache map[string]*store.DomainBaseline) (VerifyResponse, error) {
	email, domain, err := verifier.NormalizeEmail(rawEmail)
	if err != nil {
		now := time.Now().Unix()
		rec := &store.VerificationRecord{
			ID:              uuid.NewString(),
			Email:           strings.ToLower(strings.TrimSpace(rawEmail)),
			Domain:          domain,
			Classification:  "undeliverable",
			ConfidenceScore: 100,
			RiskLevel:       "high",
			Deterministic:   true,
			State:           "completed",
			ReasonCodes:     store.JSONStrings{"syntax_invalid"},
			ProtocolSummary: err.Error(),
			ExpiresAt:       now + int64(s.cfg.UndeliverableTTL.Seconds()),
			LastVerifiedAt:  now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		return responseFromRecord(rec, false), nil
	}

	userID := ""
	if user != nil {
		userID = user.ID
	}

	now := time.Now().Unix()
	existing, err := s.repo.GetByEmailAndUser(ctx, email, userID)
	if err != nil {
		return VerifyResponse{}, err
	}
	if existing != nil && existing.ExpiresAt > now {
		return responseFromRecord(existing, true), nil
	}

	recordID := uuid.NewString()
	createdAt := now
	if existing != nil {
		recordID = existing.ID
		createdAt = existing.CreatedAt
	}

	rec := &store.VerificationRecord{
		ID:             recordID,
		Email:          email,
		Domain:         domain,
		UserID:         userID,
		CreatedAt:      createdAt,
		UpdatedAt:      now,
		LastVerifiedAt: now,
	}

	routing, resolveErr := s.verifier.Resolve(ctx, domain)
	callouts := []store.VerificationCalloutAttempt{}
	if resolveErr != nil {
		rec.Classification = "undeliverable"
		rec.ConfidenceScore = 100
		rec.RiskLevel = "high"
		rec.Deterministic = true
		rec.State = "completed"
		rec.ReasonCodes = store.JSONStrings{"dns_no_mail_routing"}
		rec.ProtocolSummary = resolveErr.Error()
		rec.ExpiresAt = now + int64(s.cfg.UndeliverableTTL.Seconds())
	} else {
		target := s.verifier.CheckRecipient(ctx, routing, email)
		callouts = append(callouts, toStoreAttempts(target.Attempts)...)

		switch target.Outcome {
		case "accepted":
			baseline, baselineAttempts, err := s.getOrCreateBaseline(ctx, routing, baselineCache)
			if err != nil {
				rec.Classification = "unknown"
				rec.ConfidenceScore = 40
				rec.RiskLevel = "high"
				rec.Deterministic = false
				rec.State = "enriching"
				rec.ReasonCodes = store.JSONStrings{"baseline_lookup_failed", "recipient_accepted"}
				rec.ProtocolSummary = fmt.Sprintf("recipient accepted but baseline lookup failed: %v", err)
				rec.ExpiresAt = now + int64(s.cfg.UnknownTTL.Seconds())
			} else {
				callouts = append(callouts, baselineAttempts...)
				switch baseline.Classification {
				case "reject_unknown":
					rec.Classification = "deliverable"
					rec.ConfidenceScore = 92
					rec.RiskLevel = "low"
					rec.Deterministic = true
					rec.State = "completed"
					rec.ReasonCodes = store.JSONStrings{"recipient_accepted", "control_recipient_rejected"}
					rec.ProtocolSummary = fmt.Sprintf("recipient accepted by %s and control recipient rejected", baseline.SMTPHost)
					rec.ExpiresAt = now + int64(s.cfg.DeliverableTTL.Seconds())
				case "accept_all":
					rec.Classification = "accept_all"
					rec.ConfidenceScore = 35
					rec.RiskLevel = "high"
					rec.Deterministic = false
					rec.State = "enriching"
					rec.ReasonCodes = store.JSONStrings{"recipient_accepted", "control_recipient_accepted"}
					rec.ProtocolSummary = fmt.Sprintf("recipient accepted by %s and domain baseline accepted a random control recipient", baseline.SMTPHost)
					rec.ExpiresAt = now + int64(s.cfg.AcceptAllTTL.Seconds())
				default:
					rec.Classification = "unknown"
					rec.ConfidenceScore = 40
					rec.RiskLevel = "high"
					rec.Deterministic = false
					rec.State = "enriching"
					rec.ReasonCodes = store.JSONStrings{"recipient_accepted", "control_recipient_inconclusive"}
					rec.ProtocolSummary = "recipient accepted but control recipient baseline was inconclusive"
					rec.ExpiresAt = now + int64(s.cfg.UnknownTTL.Seconds())
				}
			}
		case "rejected":
			rec.Classification = "undeliverable"
			rec.ConfidenceScore = 100
			rec.RiskLevel = "high"
			rec.Deterministic = true
			rec.State = "completed"
			rec.ReasonCodes = store.JSONStrings{"hard_rcpt_reject"}
			rec.ProtocolSummary = target.Message
			rec.ExpiresAt = now + int64(s.cfg.UndeliverableTTL.Seconds())
		case "policy":
			rec.Classification = "unknown"
			rec.ConfidenceScore = 30
			rec.RiskLevel = "high"
			rec.Deterministic = false
			rec.State = "enriching"
			rec.ReasonCodes = store.JSONStrings{"provider_policy_block"}
			rec.ProtocolSummary = target.Message
			rec.ExpiresAt = now + int64(s.cfg.UnknownTTL.Seconds())
		case "tempfail":
			rec.Classification = "unknown"
			rec.ConfidenceScore = 25
			rec.RiskLevel = "high"
			rec.Deterministic = false
			rec.State = "enriching"
			rec.ReasonCodes = store.JSONStrings{"temporary_failure"}
			rec.ProtocolSummary = target.Message
			rec.ExpiresAt = now + int64(s.cfg.UnknownTTL.Seconds())
		default:
			rec.Classification = "unknown"
			rec.ConfidenceScore = 20
			rec.RiskLevel = "high"
			rec.Deterministic = false
			rec.State = "enriching"
			rec.ReasonCodes = store.JSONStrings{"callout_error"}
			rec.ProtocolSummary = target.Message
			rec.ExpiresAt = now + int64(s.cfg.UnknownTTL.Seconds())
		}
	}

	if err := s.repo.UpsertVerification(ctx, rec); err != nil {
		return VerifyResponse{}, err
	}
	if err := s.repo.ReplaceEnrichmentEvidence(ctx, rec.ID, nil); err != nil {
		return VerifyResponse{}, err
	}
	if err := s.repo.AddCalloutAttempts(ctx, rec.ID, callouts); err != nil {
		return VerifyResponse{}, err
	}

	if rec.State == "enriching" {
		select {
		case s.enrichmentQueue <- rec.ID:
		default:
			go func(id string) {
				select {
				case s.enrichmentQueue <- id:
				case <-ctx.Done():
				}
			}(rec.ID)
		}
	}

	return responseFromRecord(rec, false), nil
}

func (s *EmailVerificationService) getOrCreateBaseline(ctx context.Context, routing verifier.MailRouting, cache map[string]*store.DomainBaseline) (*store.DomainBaseline, []store.VerificationCalloutAttempt, error) {
	key := routing.Domain + "|" + routing.Fingerprint
	now := time.Now().Unix()

	if baseline, ok := cache[key]; ok && baseline != nil && baseline.ExpiresAt > now {
		return baseline, nil, nil
	}

	baseline, err := s.repo.GetDomainBaseline(ctx, routing.Domain, routing.Fingerprint, now)
	if err != nil {
		return nil, nil, err
	}
	if baseline != nil {
		cache[key] = baseline
		return baseline, nil, nil
	}

	controlAddress := fmt.Sprintf("definitely-not-real-%s@%s", uuid.NewString()[:12], routing.Domain)
	control := s.verifier.CheckRecipient(ctx, routing, controlAddress)
	baseline = &store.DomainBaseline{
		Domain:        routing.Domain,
		MXFingerprint: routing.Fingerprint,
		SampleAddress: controlAddress,
		CheckedAt:     now,
		ExpiresAt:     now + int64(s.cfg.DomainBaselineTTL.Seconds()),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if len(control.Attempts) > 0 {
		last := control.Attempts[len(control.Attempts)-1]
		baseline.SMTPHost = last.Host
		baseline.SMTPCode = last.Code
		baseline.SMTPMessage = last.Message
	}

	switch control.Outcome {
	case "accepted":
		baseline.Classification = "accept_all"
	case "rejected":
		baseline.Classification = "reject_unknown"
	default:
		baseline.Classification = "inconclusive"
	}

	if err := s.repo.UpsertDomainBaseline(ctx, baseline); err != nil {
		return nil, nil, err
	}
	cache[key] = baseline

	return baseline, toStoreAttempts(control.Attempts), nil
}

func (s *EmailVerificationService) enrichVerification(ctx context.Context, verificationID string) error {
	record, err := s.repo.GetVerificationByID(ctx, verificationID)
	if err != nil {
		return err
	}
	if record == nil || record.State != "enriching" {
		return nil
	}

	result, err := s.enricher.Enrich(ctx, record)
	if err != nil {
		return err
	}

	if err := s.repo.ReplaceEnrichmentEvidence(ctx, record.ID, result.Evidence); err != nil {
		return err
	}
	if err := s.repo.UpdateVerificationEnrichment(ctx, record.ID, result.ConfidenceScore, result.RiskLevel, "completed", result.Summary); err != nil {
		return err
	}
	return nil
}

func (s *EmailVerificationService) ListVerifications(ctx context.Context, userID string, limit, offset int) ([]store.VerificationRecord, error) {
	return s.repo.ListVerificationsByUser(ctx, userID, limit, offset)
}

func (s *EmailVerificationService) GetVerification(ctx context.Context, id string) (*store.VerificationRecord, error) {
	return s.repo.GetVerificationByID(ctx, id)
}

func (s *EmailVerificationService) GetVerificationDetail(ctx context.Context, id string) (*VerifyResponse, error) {
	record, err := s.repo.GetVerificationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}

	evidence, err := s.repo.ListEnrichmentEvidence(ctx, id)
	if err != nil {
		return nil, err
	}
	callouts, err := s.repo.ListCalloutAttempts(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := responseFromRecord(record, false)
	resp.Evidence = evidence
	resp.Callouts = callouts
	return &resp, nil
}

func (s *EmailVerificationService) GetVerificationStats(ctx context.Context, userID string) (map[string]int, error) {
	return s.repo.GetVerificationStats(ctx, userID)
}

func (s *EmailVerificationService) ListAllVerifications(ctx context.Context, limit, offset int) ([]store.VerificationRecord, error) {
	return s.repo.ListAllVerifications(ctx, limit, offset)
}

func (s *EmailVerificationService) DeleteVerificationForUser(ctx context.Context, id, userID string) error {
	record, err := s.repo.GetVerificationByID(ctx, id)
	if err != nil {
		return err
	}
	if record == nil {
		return ErrVerificationNotFound
	}
	if record.UserID != userID {
		return ErrVerificationForbidden
	}
	return s.repo.DeleteVerification(ctx, id)
}

func (s *EmailVerificationService) AdminDeleteVerification(ctx context.Context, id string) error {
	return s.repo.DeleteVerification(ctx, id)
}

func responseFromRecord(rec *store.VerificationRecord, cached bool) VerifyResponse {
	return VerifyResponse{
		ID:                rec.ID,
		Email:             rec.Email,
		Domain:            rec.Domain,
		Classification:    rec.Classification,
		ConfidenceScore:   rec.ConfidenceScore,
		RiskLevel:         rec.RiskLevel,
		Deterministic:     rec.Deterministic,
		State:             rec.State,
		ReasonCodes:       append([]string(nil), rec.ReasonCodes...),
		ProtocolSummary:   rec.ProtocolSummary,
		EnrichmentSummary: rec.EnrichmentSummary,
		ExpiresAt:         rec.ExpiresAt,
		LastVerifiedAt:    rec.LastVerifiedAt,
		LastEnrichedAt:    rec.LastEnrichedAt,
		Cached:            cached,
	}
}

func toStoreAttempts(attempts []verifier.CalloutAttempt) []store.VerificationCalloutAttempt {
	items := make([]store.VerificationCalloutAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		items = append(items, store.VerificationCalloutAttempt{
			SMTPHost:    attempt.Host,
			SMTPPort:    attempt.Port,
			Stage:       attempt.Stage,
			Recipient:   attempt.Recipient,
			Outcome:     attempt.Outcome,
			SMTPCode:    attempt.Code,
			SMTPMessage: attempt.Message,
			DurationMS:  attempt.DurationMS,
		})
	}
	return items
}
