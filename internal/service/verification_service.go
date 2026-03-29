package service

import (
	"context"
	"email-verifier-api/internal/repo"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"email-verifier-api/internal/store"
	"email-verifier-api/internal/verifier"

	"github.com/google/uuid"
)

type ServiceConfig struct {
	FirstBounceDelay  time.Duration
	SecondBounceDelay time.Duration
	CheckInterval     time.Duration
}

type VerifyResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Source      string `json:"source"`
	Cached      bool   `json:"cached"`
	Finalized   bool   `json:"finalized"`
	NextCheckAt int64  `json:"next_check_at,omitempty"`
}

type EmailVerificationService struct {
	verifier      *verifier.EmailVerifier
	repo          *repo.Repository
	probeSender   *SMTPProbeSender
	bounceChecker *IMAPBounceChecker
	webhook       WebhookDispatcher
	cfg           ServiceConfig
}

type SMTPAccountCreateRequest struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Sender      string `json:"sender"`
	IMAPHost    string `json:"imap_host"`
	IMAPPort    int    `json:"imap_port"`
	IMAPMailbox string `json:"imap_mailbox"`
	DailyLimit  int    `json:"daily_limit"`
	Active      *bool  `json:"active"`
}

type EmailTemplateCreateRequest struct {
	Name            string `json:"name"`
	SubjectTemplate string `json:"subject_template"`
	BodyTemplate    string `json:"body_template"`
	Active          *bool  `json:"active"`
}

func NewEmailVerificationService(
	v *verifier.EmailVerifier,
	r *repo.Repository,
	probeSender *SMTPProbeSender,
	bounceChecker *IMAPBounceChecker,
	webhook WebhookDispatcher,
	cfg ServiceConfig,
) *EmailVerificationService {
	return &EmailVerificationService{
		verifier:      v,
		repo:          r,
		probeSender:   probeSender,
		bounceChecker: bounceChecker,
		webhook:       webhook,
		cfg:           cfg,
	}
}

func (s *EmailVerificationService) VerifyEmail(ctx context.Context, email string) (VerifyResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return VerifyResponse{}, fmt.Errorf("email is required")
	}

	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return VerifyResponse{}, err
	}
	if existing != nil {
		return responseFromRecord(existing, true), nil
	}

	now := time.Now().Unix()
	directResult := s.verifier.Verify(email)
	record := &store.VerificationRecord{
		ID:             uuid.NewString(),
		Email:          email,
		Status:         directResult.Status,
		Message:        directResult.Message,
		Source:         "direct-smtp-check",
		FirstCheckedAt: now,
		LastCheckedAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	requiresFallback := directResult.Status == "error" || directResult.Status == "unknown" || directResult.Status == "greylisted"
	if requiresFallback {
		token := uuid.NewString()
		record.ProbeToken = token
		record.Source = "smtp-probe"

		accountID, err := s.probeSender.SendProbe(ctx, email, token)
		if err != nil {
			record.Status = "error"
			record.Message = fmt.Sprintf("fallback probe send failed: %v", err)
			record.Finalized = true
		} else {
			record.SMTPAccountID = accountID
			record.Status = "pending_bounce_check"
			record.Message = fmt.Sprintf("probe sent via smtp account %s; single bounce check scheduled at 6h", accountID)
			record.NextCheckAt = time.Now().Add(s.cfg.SecondBounceDelay).Unix()
			record.Finalized = false
		}
	} else {
		record.Finalized = true
	}

	if err := s.repo.UpsertVerification(ctx, record); err != nil {
		return VerifyResponse{}, err
	}
	if err := s.repo.AddEvent(ctx, record.ID, "verify.requested", record.Status, record.Message); err != nil {
		log.Printf("warning: failed to save event: %v", err)
	}

	if err := s.webhook.Send(ctx, "verify.created", record); err != nil {
		log.Printf("warning: webhook failed: %v", err)
	}

	return responseFromRecord(record, false), nil
}

func (s *EmailVerificationService) CreateSMTPAccount(ctx context.Context, req SMTPAccountCreateRequest) (*store.SMTPAccount, error) {
	req.Host = strings.TrimSpace(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.Sender = strings.TrimSpace(req.Sender)
	req.IMAPHost = strings.TrimSpace(req.IMAPHost)
	req.IMAPMailbox = strings.TrimSpace(req.IMAPMailbox)

	if req.Host == "" || req.Username == "" || req.Password == "" || req.Sender == "" {
		return nil, errors.New("host, username, password, and sender are required")
	}
	if req.IMAPHost == "" {
		req.IMAPHost = req.Host
	}
	if req.Port == 0 {
		req.Port = 587
	}
	if req.IMAPPort == 0 {
		req.IMAPPort = 993
	}
	if req.IMAPMailbox == "" {
		req.IMAPMailbox = "INBOX"
	}
	if req.DailyLimit <= 0 {
		req.DailyLimit = 100
	}

	input := store.SMTPAccountInput{
		ID:          uuid.NewString(),
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		Password:    req.Password,
		Sender:      req.Sender,
		IMAPHost:    req.IMAPHost,
		IMAPPort:    req.IMAPPort,
		IMAPMailbox: req.IMAPMailbox,
		DailyLimit:  req.DailyLimit,
		Active:      true,
	}
	if req.Active != nil {
		input.Active = *req.Active
	}

	return s.repo.CreateSMTPAccount(ctx, input)
}

func (s *EmailVerificationService) ListSMTPAccounts(ctx context.Context) ([]store.SMTPAccount, error) {
	return s.repo.ListSMTPAccounts(ctx)
}

func (s *EmailVerificationService) CreateEmailTemplate(ctx context.Context, req EmailTemplateCreateRequest) (*store.EmailTemplate, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.SubjectTemplate = strings.TrimSpace(req.SubjectTemplate)
	req.BodyTemplate = strings.TrimSpace(req.BodyTemplate)
	if req.Name == "" || req.SubjectTemplate == "" || req.BodyTemplate == "" {
		return nil, errors.New("name, subject_template, and body_template are required")
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	input := store.EmailTemplateInput{
		ID:              uuid.NewString(),
		Name:            req.Name,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
		Active:          active,
	}

	return s.repo.CreateEmailTemplate(ctx, input)
}

func (s *EmailVerificationService) ListEmailTemplates(ctx context.Context) ([]store.EmailTemplate, error) {
	return s.repo.ListEmailTemplates(ctx)
}

func (s *EmailVerificationService) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.ProcessDueChecks(ctx); err != nil {
				log.Printf("warning: process due checks failed: %v", err)
			}
		}
	}
}

func (s *EmailVerificationService) ProcessDueChecks(ctx context.Context) error {
	now := time.Now().Unix()
	due, err := s.repo.ListDueChecks(ctx, now, 100)
	if err != nil {
		return err
	}

	for i := range due {
		rec := due[i]
		if err := s.processOneDue(ctx, &rec); err != nil {
			log.Printf("warning: process due check failed for %s: %v", rec.Email, err)
		}
	}

	return nil
}

func (s *EmailVerificationService) processOneDue(ctx context.Context, rec *store.VerificationRecord) error {
	if rec.SMTPAccountID == "" {
		return fmt.Errorf("missing smtp account reference on verification record")
	}

	account, err := s.repo.GetSMTPAccountByID(ctx, rec.SMTPAccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("smtp account not found: %s", rec.SMTPAccountID)
	}

	bounced, reason, err := s.bounceChecker.HasBounce(ctx, IMAPConfig{
		Host:     account.IMAPHost,
		Port:     account.IMAPPort,
		Username: account.Username,
		Password: account.Password,
		Mailbox:  account.IMAPMailbox,
	}, rec.Email, rec.ProbeToken)

	now := time.Now().Unix()
	rec.LastCheckedAt = now
	rec.UpdatedAt = now
	rec.CheckCount++
	rec.NextCheckAt = 0
	rec.Finalized = true

	event := "verify.check.no_bounce"
	if err != nil {
		rec.Message = fmt.Sprintf("single bounce check error: %v", err)
		event = "verify.check.error"
	} else if bounced {
		rec.Status = "bounced"
		rec.Message = reason
		event = "verify.bounced"
	} else {
		rec.Message = "no bounce detected in single scheduled check; keeping existing status"
	}

	if err := s.repo.UpsertVerification(ctx, rec); err != nil {
		return err
	}
	if err := s.repo.AddEvent(ctx, rec.ID, event, rec.Status, rec.Message); err != nil {
		log.Printf("warning: failed to save event: %v", err)
	}
	if err := s.webhook.Send(ctx, event, rec); err != nil {
		log.Printf("warning: webhook failed: %v", err)
	}

	return nil
}

func responseFromRecord(rec *store.VerificationRecord, cached bool) VerifyResponse {
	resp := VerifyResponse{
		ID:        rec.ID,
		Email:     rec.Email,
		Status:    rec.Status,
		Message:   rec.Message,
		Source:    rec.Source,
		Cached:    cached,
		Finalized: rec.Finalized,
	}
	if rec.NextCheckAt > 0 {
		resp.NextCheckAt = rec.NextCheckAt
	}
	return resp
}
