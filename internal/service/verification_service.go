package service

import (
	"context"
	"email-verifier-api/internal/repo"
	"email-verifier-api/internal/serviceutil"
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
	HardResultTTL     time.Duration
	DirectValidTTL    time.Duration
	ProbeValidTTL     time.Duration
	TransientTTL      time.Duration
}

var (
	ErrVerificationNotFound  = errors.New("verification not found")
	ErrVerificationForbidden = errors.New("verification access denied")
)

type VerifyResponse struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	Source           string `json:"source"`
	Cached           bool   `json:"cached"`
	Finalized        bool   `json:"finalized"`
	NextCheckAt      int64  `json:"next_check_at,omitempty"`
	Confidence       string `json:"confidence"`
	Deterministic    bool   `json:"deterministic"`
	ReasonCode       string `json:"reason_code"`
	VerificationPath string `json:"verification_path"`
	SignalSummary    string `json:"signal_summary"`
	ExpiresAt        int64  `json:"expires_at"`
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
	if cfg.HardResultTTL <= 0 {
		cfg.HardResultTTL = 7 * 24 * time.Hour
	}
	if cfg.DirectValidTTL <= 0 {
		cfg.DirectValidTTL = 72 * time.Hour
	}
	if cfg.ProbeValidTTL <= 0 {
		cfg.ProbeValidTTL = 24 * time.Hour
	}
	if cfg.TransientTTL <= 0 {
		cfg.TransientTTL = 6 * time.Hour
	}

	return &EmailVerificationService{
		verifier:      v,
		repo:          r,
		probeSender:   probeSender,
		bounceChecker: bounceChecker,
		webhook:       webhook,
		cfg:           cfg,
	}
}

func (s *EmailVerificationService) VerifyEmail(ctx context.Context, email string, user *store.User) (VerifyResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return VerifyResponse{}, fmt.Errorf("email is required")
	}

	userID := ""
	if user != nil {
		userID = user.ID
	}

	existing, err := s.repo.GetByEmailAndUser(ctx, email, userID)
	if err != nil {
		return VerifyResponse{}, err
	}
	now := time.Now().Unix()
	if existing != nil {
		if !existing.Finalized && existing.NextCheckAt > 0 {
			return responseFromRecord(existing, true), nil
		}
		if existing.ExpiresAt > now {
			return responseFromRecord(existing, true), nil
		}
	}

	recordID := uuid.NewString()
	createdAt := now
	firstCheckedAt := now
	checkCount := 0
	if existing != nil {
		recordID = existing.ID
		createdAt = existing.CreatedAt
		firstCheckedAt = existing.FirstCheckedAt
		checkCount = existing.CheckCount
	}

	directResult := s.verifier.Verify(email)
	record := &store.VerificationRecord{
		ID:               recordID,
		Email:            email,
		UserID:           userID,
		Status:           directResult.Status,
		Message:          directResult.Message,
		Source:           "direct-smtp-check",
		Confidence:       "low",
		Deterministic:    directResult.Deterministic,
		ReasonCode:       directResult.ReasonCode,
		VerificationPath: "direct_smtp",
		SignalSummary:    serviceutil.SummarizeLocalSignals(email, directResult.SignalSummary),
		ExpiresAt:        now + int64(s.cfg.TransientTTL.Seconds()),
		CheckCount:       checkCount,
		Finalized:        true,
		FirstCheckedAt:   firstCheckedAt,
		LastCheckedAt:    now,
		CreatedAt:        createdAt,
		UpdatedAt:        now,
	}

	s.applyDirectMetadata(record, directResult)

	requiresFallback := directResult.RequireProbe || directResult.Status == "error" || directResult.Status == "unknown" || directResult.Status == "greylisted"
	if requiresFallback {
		token := uuid.NewString()
		record.ProbeToken = token
		record.Source = "smtp-probe"

		accountID, err := s.probeSender.SendProbeForUser(ctx, email, token, userID)
		if err != nil {
			record.Status = "error"
			record.Message = fmt.Sprintf("probe send failed: %v", err)
			record.Confidence = "low"
			record.Deterministic = false
			record.ReasonCode = "probe_send_failed"
			record.VerificationPath = serviceutil.ProbePathForReason(directResult.ReasonCode)
			record.SignalSummary = serviceutil.SummarizeLocalSignals(email, "Direct SMTP was inconclusive and the fallback probe could not be sent.")
			record.ExpiresAt = now + int64(s.cfg.TransientTTL.Seconds())
			record.Finalized = true
		} else {
			record.SMTPAccountID = accountID
			record.Status = "pending_bounce_check"
			record.Message = fmt.Sprintf("probe sent via smtp account %s; waiting for bounce window", accountID)
			record.Confidence = "low"
			record.Deterministic = false
			record.ReasonCode = "probe_sent_waiting_bounce"
			record.VerificationPath = serviceutil.ProbePathForReason(directResult.ReasonCode)
			record.SignalSummary = serviceutil.SummarizeLocalSignals(email, probeQueuedSummary(record, accountID))
			record.NextCheckAt = time.Now().Add(s.cfg.FirstBounceDelay).Unix()
			record.ExpiresAt = now + int64(s.cfg.TransientTTL.Seconds())
			record.Finalized = false
		}
	}

	if err := s.repo.UpsertVerification(ctx, record); err != nil {
		return VerifyResponse{}, err
	}
	if err := s.repo.AddEvent(ctx, record.ID, "verify.requested", record.Status, record.Message); err != nil {
		log.Printf("warning: failed to save event: %v", err)
	}

	webhookURL := ""
	if user != nil && user.WebhookURL != "" {
		webhookURL = user.WebhookURL
	}
	if err := s.webhook.SendWithURL(ctx, "verify.created", record, webhookURL); err != nil {
		log.Printf("warning: webhook failed: %v", err)
	}

	return responseFromRecord(record, false), nil
}

func (s *EmailVerificationService) VerifyEmailBatch(ctx context.Context, emails []string, user *store.User) ([]VerifyResponse, int) {
	responses := make([]VerifyResponse, 0, len(emails))
	accepted := 0

	for _, rawEmail := range emails {
		email := strings.TrimSpace(rawEmail)
		result, err := s.VerifyEmail(ctx, email, user)
		if err != nil {
			responses = append(responses, VerifyResponse{
				Email:            strings.ToLower(email),
				Status:           "error",
				Message:          err.Error(),
				Source:           "batch-api",
				Cached:           false,
				Finalized:        true,
				Confidence:       "low",
				Deterministic:    false,
				ReasonCode:       "internal_error",
				VerificationPath: "direct_smtp",
				SignalSummary:    "Batch verification failed before a verification record could be stored.",
			})
			continue
		}

		responses = append(responses, result)
		accepted++
	}

	return responses, accepted
}

func (s *EmailVerificationService) CreateSMTPAccount(ctx context.Context, req SMTPAccountCreateRequest, userID string) (*store.SMTPAccount, error) {
	req.Host = serviceutil.NormalizeServerHost(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.Sender = strings.TrimSpace(req.Sender)
	req.IMAPHost = serviceutil.NormalizeServerHost(req.IMAPHost)
	req.IMAPMailbox = strings.TrimSpace(req.IMAPMailbox)

	if req.Host == "" || req.Username == "" || req.Password == "" || req.Sender == "" {
		return nil, errors.New("host, username, password, and sender are required")
	}
	if req.IMAPHost == "" {
		req.IMAPHost = serviceutil.InferIMAPHost(req.Host)
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
	if err := serviceutil.ValidateServerHost("host", req.Host); err != nil {
		return nil, err
	}
	if err := serviceutil.ValidateServerHost("imap_host", req.IMAPHost); err != nil {
		return nil, err
	}
	if err := serviceutil.ValidatePort("port", req.Port); err != nil {
		return nil, err
	}
	if err := serviceutil.ValidatePort("imap_port", req.IMAPPort); err != nil {
		return nil, err
	}

	input := store.SMTPAccountInput{
		ID:          uuid.NewString(),
		UserID:      userID,
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

func (s *EmailVerificationService) ListSMTPAccounts(ctx context.Context, userID string) ([]store.SMTPAccount, error) {
	if userID == "" {
		return s.repo.ListSMTPAccounts(ctx)
	}
	return s.repo.ListSMTPAccountsByUser(ctx, userID)
}

func (s *EmailVerificationService) CreateEmailTemplate(ctx context.Context, req EmailTemplateCreateRequest, userID string) (*store.EmailTemplate, error) {
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
		UserID:          userID,
		Name:            req.Name,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
		Active:          active,
	}

	return s.repo.CreateEmailTemplate(ctx, input)
}

func (s *EmailVerificationService) ListEmailTemplates(ctx context.Context, userID string) ([]store.EmailTemplate, error) {
	if userID == "" {
		return s.repo.ListEmailTemplates(ctx)
	}
	return s.repo.ListEmailTemplatesByUser(ctx, userID)
}

func (s *EmailVerificationService) GetEmailTemplate(ctx context.Context, id string) (*store.EmailTemplate, error) {
	return s.repo.GetEmailTemplateByID(ctx, id)
}

func (s *EmailVerificationService) UpdateEmailTemplate(ctx context.Context, id string, req EmailTemplateCreateRequest, userID string) (*store.EmailTemplate, error) {
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
		ID:              id,
		UserID:          userID,
		Name:            req.Name,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
		Active:          active,
	}

	return s.repo.UpdateEmailTemplate(ctx, id, input)
}

func (s *EmailVerificationService) DeleteEmailTemplate(ctx context.Context, id string) error {
	return s.repo.DeleteEmailTemplate(ctx, id)
}

func (s *EmailVerificationService) GetSMTPAccount(ctx context.Context, id string) (*store.SMTPAccount, error) {
	return s.repo.GetSMTPAccountByID(ctx, id)
}

func (s *EmailVerificationService) UpdateSMTPAccount(ctx context.Context, id string, req SMTPAccountCreateRequest, userID string) (*store.SMTPAccount, error) {
	req.Host = serviceutil.NormalizeServerHost(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.Sender = strings.TrimSpace(req.Sender)
	req.IMAPHost = serviceutil.NormalizeServerHost(req.IMAPHost)
	req.IMAPMailbox = strings.TrimSpace(req.IMAPMailbox)

	if req.Host == "" || req.Username == "" || req.Sender == "" {
		return nil, errors.New("host, username, and sender are required")
	}
	if req.IMAPHost == "" {
		req.IMAPHost = serviceutil.InferIMAPHost(req.Host)
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
	if err := serviceutil.ValidateServerHost("host", req.Host); err != nil {
		return nil, err
	}
	if err := serviceutil.ValidateServerHost("imap_host", req.IMAPHost); err != nil {
		return nil, err
	}
	if err := serviceutil.ValidatePort("port", req.Port); err != nil {
		return nil, err
	}
	if err := serviceutil.ValidatePort("imap_port", req.IMAPPort); err != nil {
		return nil, err
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	input := store.SMTPAccountInput{
		ID:          id,
		UserID:      userID,
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		Password:    req.Password,
		Sender:      req.Sender,
		IMAPHost:    req.IMAPHost,
		IMAPPort:    req.IMAPPort,
		IMAPMailbox: req.IMAPMailbox,
		DailyLimit:  req.DailyLimit,
		Active:      active,
	}

	return s.repo.UpdateSMTPAccount(ctx, id, input)
}

func (s *EmailVerificationService) DeleteSMTPAccount(ctx context.Context, id string) error {
	return s.repo.DeleteSMTPAccount(ctx, id)
}

func (s *EmailVerificationService) ListVerifications(ctx context.Context, userID string, limit, offset int) ([]store.VerificationRecord, error) {
	return s.repo.ListVerificationsByUser(ctx, userID, limit, offset)
}

func (s *EmailVerificationService) GetVerification(ctx context.Context, id string) (*store.VerificationRecord, error) {
	return s.repo.GetVerificationByID(ctx, id)
}

func (s *EmailVerificationService) GetVerificationStats(ctx context.Context, userID string) (map[string]int, error) {
	return s.repo.GetVerificationStats(ctx, userID)
}

func (s *EmailVerificationService) ListAllVerifications(ctx context.Context, limit, offset int) ([]store.VerificationRecord, error) {
	return s.repo.ListAllVerifications(ctx, limit, offset)
}

func (s *EmailVerificationService) StartScheduler(ctx context.Context) {
	if count, err := s.repo.ResetSMTPDailyUsage(ctx); err != nil {
		log.Printf("warning: initial smtp daily reset failed: %v", err)
	} else if count > 0 {
		log.Printf("info: initial reset of smtp daily usage counters for %d account(s)", count)
	}

	bounceTicker := time.NewTicker(s.cfg.CheckInterval)
	defer bounceTicker.Stop()

	dailyResetTicker := time.NewTicker(24 * time.Hour)
	defer dailyResetTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dailyResetTicker.C:
			if count, err := s.repo.ResetSMTPDailyUsage(ctx); err != nil {
				log.Printf("warning: smtp daily reset failed: %v", err)
			} else if count > 0 {
				log.Printf("info: reset smtp daily usage counters for %d account(s)", count)
			}
		case <-bounceTicker.C:
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

	imapHost := serviceutil.NormalizeServerHost(account.IMAPHost)
	if imapHost == "" {
		imapHost = serviceutil.InferIMAPHost(account.Host)
	}

	bounced, reason, matchKind, err := s.bounceChecker.HasBounce(ctx, IMAPConfig{
		Host:     imapHost,
		Port:     account.IMAPPort,
		Username: account.Username,
		Password: account.Password,
		Mailbox:  account.IMAPMailbox,
	}, rec.Email, rec.ProbeToken)

	now := time.Now().Unix()
	checkNumber := rec.CheckCount + 1
	rec.LastCheckedAt = now
	rec.UpdatedAt = now
	rec.CheckCount++
	rec.ExpiresAt = now + int64(s.cfg.TransientTTL.Seconds())

	event := "verify.check.first.no_bounce"
	if checkNumber >= 2 {
		event = "verify.check.second.no_bounce"
	}

	if err != nil {
		if checkNumber == 1 {
			rec.Status = "pending_bounce_check"
			rec.Message = fmt.Sprintf("first IMAP bounce check failed: %v; retry scheduled", err)
			rec.Confidence = "low"
			rec.Deterministic = false
			rec.ReasonCode = "imap_check_retrying"
			rec.VerificationPath = ensureProbePath(rec.VerificationPath)
			rec.SignalSummary = serviceutil.SummarizeLocalSignals(rec.Email, "Bounce evidence was unavailable during the first IMAP check, so the verification remains pending.")
			rec.NextCheckAt = time.Now().Add(s.cfg.SecondBounceDelay).Unix()
			rec.Finalized = false
			event = "verify.check.first.error"
		} else {
			rec.Status = "error"
			rec.Message = fmt.Sprintf("second IMAP bounce check failed: %v", err)
			rec.Confidence = "low"
			rec.Deterministic = false
			rec.ReasonCode = "imap_check_failed_final"
			rec.VerificationPath = ensureProbePath(rec.VerificationPath)
			rec.SignalSummary = serviceutil.SummarizeLocalSignals(rec.Email, "Bounce verification failed across the full check window, so mailbox validity remains unresolved.")
			rec.NextCheckAt = 0
			rec.Finalized = true
			event = "verify.check.second.error"
		}
	} else if bounced {
		rec.Status = "invalid"
		rec.Message = reason
		rec.Confidence = "high"
		rec.Deterministic = true
		rec.ReasonCode = serviceutil.BounceReasonCode(matchKind)
		rec.VerificationPath = ensureProbePath(rec.VerificationPath)
		rec.SignalSummary = serviceutil.SummarizeLocalSignals(rec.Email, serviceutil.BounceSignalSummary(matchKind))
		rec.NextCheckAt = 0
		rec.ExpiresAt = now + int64(s.cfg.HardResultTTL.Seconds())
		rec.Finalized = true
		event = "verify.invalid"
	} else {
		if checkNumber == 1 {
			rec.Status = "valid"
			rec.Message = "no bounce observed in the first check window; second check scheduled"
			rec.Confidence = "low"
			rec.Deterministic = false
			rec.ReasonCode = "no_bounce_first_window"
			rec.VerificationPath = ensureProbePath(rec.VerificationPath)
			rec.SignalSummary = serviceutil.SummarizeLocalSignals(rec.Email, "No bounce has been observed in the first check window. The address is marked valid while the second bounce check remains scheduled.")
			rec.NextCheckAt = time.Now().Add(s.cfg.SecondBounceDelay).Unix()
			rec.Finalized = false
			event = "verify.check.first.no_bounce"
		} else {
			rec.Status = "valid"
			rec.Message = "no bounce observed within the configured verification window"
			rec.Confidence = "low"
			rec.Deterministic = false
			rec.ReasonCode = "no_bounce_second_window"
			rec.VerificationPath = ensureProbePath(rec.VerificationPath)
			rec.SignalSummary = serviceutil.SummarizeLocalSignals(rec.Email, "No bounce was observed across both check windows. This remains a heuristic signal rather than confirmed mailbox existence.")
			rec.NextCheckAt = 0
			rec.ExpiresAt = now + int64(s.cfg.ProbeValidTTL.Seconds())
			rec.Finalized = true
			event = "verify.check.second.no_bounce"
		}
	}

	if err := s.repo.UpsertVerification(ctx, rec); err != nil {
		return err
	}
	if err := s.repo.AddEvent(ctx, rec.ID, event, rec.Status, rec.Message); err != nil {
		log.Printf("warning: failed to save event: %v", err)
	}

	webhookURL := ""
	if rec.UserID != "" {
		user, err := s.repo.GetUserByID(ctx, rec.UserID)
		if err == nil && user != nil && user.WebhookURL != "" {
			webhookURL = user.WebhookURL
		}
	}
	if err := s.webhook.SendWithURL(ctx, event, rec, webhookURL); err != nil {
		log.Printf("warning: webhook failed: %v", err)
	}

	return nil
}

func responseFromRecord(rec *store.VerificationRecord, cached bool) VerifyResponse {
	resp := VerifyResponse{
		ID:               rec.ID,
		Email:            rec.Email,
		Status:           rec.Status,
		Message:          rec.Message,
		Source:           rec.Source,
		Cached:           cached,
		Finalized:        rec.Finalized,
		Confidence:       rec.Confidence,
		Deterministic:    rec.Deterministic,
		ReasonCode:       rec.ReasonCode,
		VerificationPath: rec.VerificationPath,
		SignalSummary:    rec.SignalSummary,
		ExpiresAt:        rec.ExpiresAt,
	}
	if rec.NextCheckAt > 0 {
		resp.NextCheckAt = rec.NextCheckAt
	}
	return resp
}

func (s *EmailVerificationService) DeleteVerificationForUser(ctx context.Context, id, userID string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("verification id is required")
	}

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

func (s *EmailVerificationService) SendTestWebhook(ctx context.Context, webhookURL string) error {
	now := time.Now().Unix()
	testRecord := &store.VerificationRecord{
		ID:               "test-" + uuid.NewString()[:8],
		Email:            "test@example.com",
		Status:           "valid",
		Message:          "This is a test webhook notification",
		Source:           "test",
		Confidence:       "low",
		Deterministic:    false,
		ReasonCode:       "webhook_test",
		VerificationPath: "probe_bounce",
		SignalSummary:    "Example payload showing additive evidence fields.",
		ExpiresAt:        now + int64(s.cfg.TransientTTL.Seconds()),
		FirstCheckedAt:   now,
		LastCheckedAt:    now,
		Finalized:        true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return s.webhook.SendWithURL(ctx, "test.webhook", testRecord, webhookURL)
}

func (s *EmailVerificationService) applyDirectMetadata(rec *store.VerificationRecord, result verifier.VerifyResult) {
	rec.VerificationPath = "direct_smtp"
	rec.SignalSummary = serviceutil.SummarizeLocalSignals(rec.Email, result.SignalSummary)

	switch rec.Status {
	case "invalid":
		rec.Confidence = "high"
		rec.Deterministic = true
		rec.ExpiresAt = rec.LastCheckedAt + int64(s.cfg.HardResultTTL.Seconds())
	case "disposable":
		rec.Confidence = "high"
		rec.Deterministic = true
		rec.ExpiresAt = rec.LastCheckedAt + int64(s.cfg.HardResultTTL.Seconds())
	case "valid":
		rec.Confidence = "medium"
		rec.Deterministic = false
		rec.ExpiresAt = rec.LastCheckedAt + int64(s.cfg.DirectValidTTL.Seconds())
	case "greylisted", "unknown", "error":
		rec.Confidence = "low"
		rec.Deterministic = false
		rec.ExpiresAt = rec.LastCheckedAt + int64(s.cfg.TransientTTL.Seconds())
	default:
		rec.Confidence = "low"
		rec.Deterministic = false
		rec.ExpiresAt = rec.LastCheckedAt + int64(s.cfg.TransientTTL.Seconds())
	}
}

func probeQueuedSummary(rec *store.VerificationRecord, accountID string) string {
	return fmt.Sprintf("Direct SMTP evidence was insufficient, so a probe was sent via SMTP account %s and the system is waiting for bounce evidence.", accountID)
}

func ensureProbePath(path string) string {
	if path == "" || path == "direct_smtp" {
		return "hybrid"
	}
	return path
}
