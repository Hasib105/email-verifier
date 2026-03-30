package service

import (
	"context"
	"testing"
	"time"

	"email-verifier-api/internal/store"
)

// MockVerifier is a mock email verifier for testing
type MockVerifier struct {
	VerifyFunc func(email string) VerifyResult
}

type VerifyResult struct {
	Status  string
	Message string
}

func (m *MockVerifier) Verify(email string) VerifyResult {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(email)
	}
	return VerifyResult{Status: "valid", Message: "OK"}
}

// MockRepository is a mock repository for testing
type MockRepository struct {
	verifications      map[string]*store.VerificationRecord
	smtpAccounts       map[string]*store.SMTPAccount
	emailTemplates     map[string]*store.EmailTemplate
	users              map[string]*store.User
	usersByEmail       map[string]*store.User
	usersByAPIKey      map[string]*store.User
	events             []VerificationEvent
	dueChecks          []store.VerificationRecord
	acquiredAccountIdx int
}

type VerificationEvent struct {
	VerificationID string
	EventType      string
	Status         string
	Message        string
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		verifications:  make(map[string]*store.VerificationRecord),
		smtpAccounts:   make(map[string]*store.SMTPAccount),
		emailTemplates: make(map[string]*store.EmailTemplate),
		users:          make(map[string]*store.User),
		events:         []VerificationEvent{},
		dueChecks:      []store.VerificationRecord{},
	}
}

func (r *MockRepository) GetByEmail(ctx context.Context, email string) (*store.VerificationRecord, error) {
	if rec, ok := r.verifications[email]; ok {
		return rec, nil
	}
	return nil, nil
}

func (r *MockRepository) GetByEmailAndUser(ctx context.Context, email, userID string) (*store.VerificationRecord, error) {
	key := email + ":" + userID
	if rec, ok := r.verifications[key]; ok {
		return rec, nil
	}
	return nil, nil
}

func (r *MockRepository) UpsertVerification(ctx context.Context, rec *store.VerificationRecord) error {
	key := rec.Email + ":" + rec.UserID
	r.verifications[key] = rec
	return nil
}

func (r *MockRepository) AddEvent(ctx context.Context, verificationID, eventType, status, message string) error {
	r.events = append(r.events, VerificationEvent{
		VerificationID: verificationID,
		EventType:      eventType,
		Status:         status,
		Message:        message,
	})
	return nil
}

func (r *MockRepository) ListDueChecks(ctx context.Context, nowUnix int64, limit int) ([]store.VerificationRecord, error) {
	return r.dueChecks, nil
}

func (r *MockRepository) GetSMTPAccountByID(ctx context.Context, id string) (*store.SMTPAccount, error) {
	if acc, ok := r.smtpAccounts[id]; ok {
		return acc, nil
	}
	return nil, nil
}

func (r *MockRepository) AcquireSMTPAccountForSend(ctx context.Context) (*store.SMTPAccount, error) {
	for _, acc := range r.smtpAccounts {
		if acc.Active && acc.SentToday < acc.DailyLimit {
			return acc, nil
		}
	}
	return nil, nil
}

func (r *MockRepository) AcquireSMTPAccountForSendByUser(ctx context.Context, userID string) (*store.SMTPAccount, error) {
	for _, acc := range r.smtpAccounts {
		if acc.Active && acc.SentToday < acc.DailyLimit && acc.UserID == userID {
			return acc, nil
		}
	}
	return nil, nil
}

func (r *MockRepository) GetActiveEmailTemplate(ctx context.Context) (*store.EmailTemplate, error) {
	for _, tmpl := range r.emailTemplates {
		if tmpl.Active {
			return tmpl, nil
		}
	}
	return nil, nil
}

func (r *MockRepository) GetActiveEmailTemplateByUser(ctx context.Context, userID string) (*store.EmailTemplate, error) {
	for _, tmpl := range r.emailTemplates {
		if tmpl.Active && tmpl.UserID == userID {
			return tmpl, nil
		}
	}
	return nil, nil
}

func (r *MockRepository) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, nil
}

func (r *MockRepository) AddSMTPAccount(acc *store.SMTPAccount) {
	r.smtpAccounts[acc.ID] = acc
}

func (r *MockRepository) AddUser(user *store.User) {
	r.users[user.ID] = user
}

func (r *MockRepository) AddDueCheck(rec store.VerificationRecord) {
	r.dueChecks = append(r.dueChecks, rec)
}

// MockWebhookDispatcher is a mock webhook dispatcher for testing
type MockWebhookDispatcher struct {
	SentEvents []WebhookEvent
}

type WebhookEvent struct {
	Event      string
	Record     *store.VerificationRecord
	WebhookURL string
}

func (w *MockWebhookDispatcher) Send(ctx context.Context, event string, rec *store.VerificationRecord) error {
	return w.SendWithURL(ctx, event, rec, "")
}

func (w *MockWebhookDispatcher) SendWithURL(ctx context.Context, event string, rec *store.VerificationRecord, webhookURL string) error {
	w.SentEvents = append(w.SentEvents, WebhookEvent{
		Event:      event,
		Record:     rec,
		WebhookURL: webhookURL,
	})
	return nil
}

// MockProbeSender is a mock probe sender for testing
type MockProbeSender struct {
	SendProbeFunc func(ctx context.Context, targetEmail, token, userID string) (string, error)
}

func (p *MockProbeSender) SendProbeForUser(ctx context.Context, targetEmail, token, userID string) (string, error) {
	if p.SendProbeFunc != nil {
		return p.SendProbeFunc(ctx, targetEmail, token, userID)
	}
	return "mock-account-id", nil
}

// MockBounceChecker is a mock bounce checker for testing
type MockBounceChecker struct {
	HasBounceFunc func(ctx context.Context, cfg IMAPConfig, targetEmail, token string) (bool, string, error)
}

func (b *MockBounceChecker) HasBounce(ctx context.Context, cfg IMAPConfig, targetEmail, token string) (bool, string, error) {
	if b.HasBounceFunc != nil {
		return b.HasBounceFunc(ctx, cfg, targetEmail, token)
	}
	return false, "", nil
}

// TestDirectSMTPVerification tests the direct SMTP verification flow
func TestDirectSMTPVerification_Valid(t *testing.T) {
	mockRepo := NewMockRepository()
	mockWebhook := &MockWebhookDispatcher{}

	// Add a test user
	testUser := &store.User{
		ID:         "test-user-id",
		Name:       "Test User",
		Email:      "testuser@example.com",
		APIKey:     "test-api-key",
		WebhookURL: "https://webhook.example.com/callback",
		Active:     true,
	}
	mockRepo.AddUser(testUser)

	// Create a verification service with mocks
	cfg := ServiceConfig{
		FirstBounceDelay:  2 * time.Minute,
		SecondBounceDelay: 6 * time.Hour,
		CheckInterval:     1 * time.Minute,
	}

	// Simulate direct SMTP check returning valid
	result, err := simulateVerification(mockRepo, mockWebhook, cfg, "valid@example.com", testUser, "valid")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Status != "valid" {
		t.Errorf("Expected status 'valid', got: %s", result.Status)
	}

	if result.Source != "direct-smtp-check" {
		t.Errorf("Expected source 'direct-smtp-check', got: %s", result.Source)
	}

	if !result.Finalized {
		t.Error("Expected finalized to be true for valid result")
	}

	// Verify webhook was called
	if len(mockWebhook.SentEvents) != 1 {
		t.Errorf("Expected 1 webhook event, got: %d", len(mockWebhook.SentEvents))
	}

	if mockWebhook.SentEvents[0].Event != "verify.created" {
		t.Errorf("Expected webhook event 'verify.created', got: %s", mockWebhook.SentEvents[0].Event)
	}

	// Verify user's webhook URL was used
	if mockWebhook.SentEvents[0].WebhookURL != testUser.WebhookURL {
		t.Errorf("Expected webhook URL '%s', got: %s", testUser.WebhookURL, mockWebhook.SentEvents[0].WebhookURL)
	}
}

// TestDirectSMTPVerification_Invalid tests direct SMTP verification for invalid emails
func TestDirectSMTPVerification_Invalid(t *testing.T) {
	mockRepo := NewMockRepository()
	mockWebhook := &MockWebhookDispatcher{}

	testUser := &store.User{
		ID:     "test-user-id",
		Active: true,
	}
	mockRepo.AddUser(testUser)

	cfg := ServiceConfig{
		FirstBounceDelay:  2 * time.Minute,
		SecondBounceDelay: 6 * time.Hour,
		CheckInterval:     1 * time.Minute,
	}

	result, err := simulateVerification(mockRepo, mockWebhook, cfg, "invalid@example.com", testUser, "invalid")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Status != "invalid" {
		t.Errorf("Expected status 'invalid', got: %s", result.Status)
	}

	if !result.Finalized {
		t.Error("Expected finalized to be true for invalid result")
	}
}

// TestProbeFallbackVerification tests the fallback probe + bounce check flow
func TestProbeFallbackVerification(t *testing.T) {
	mockRepo := NewMockRepository()
	mockWebhook := &MockWebhookDispatcher{}

	// Add SMTP account for probe sending
	smtpAccount := &store.SMTPAccount{
		ID:          "smtp-account-1",
		UserID:      "test-user-id",
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "probe@example.com",
		Password:    "password",
		Sender:      "probe@example.com",
		IMAPHost:    "imap.example.com",
		IMAPPort:    993,
		IMAPMailbox: "INBOX",
		DailyLimit:  100,
		SentToday:   0,
		Active:      true,
	}
	mockRepo.AddSMTPAccount(smtpAccount)

	testUser := &store.User{
		ID:         "test-user-id",
		WebhookURL: "https://webhook.example.com/callback",
		Active:     true,
	}
	mockRepo.AddUser(testUser)

	cfg := ServiceConfig{
		FirstBounceDelay:  2 * time.Minute,
		SecondBounceDelay: 6 * time.Hour,
		CheckInterval:     1 * time.Minute,
	}

	// Simulate direct SMTP check returning "unknown" (triggering fallback)
	result, err := simulateVerificationWithProbe(mockRepo, mockWebhook, cfg, "unknown@catchall.com", testUser, "unknown")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// When direct check returns unknown, we fall back to probe
	if result.Status != "pending_bounce_check" {
		t.Errorf("Expected status 'pending_bounce_check', got: %s", result.Status)
	}

	if result.Source != "smtp-probe" {
		t.Errorf("Expected source 'smtp-probe', got: %s", result.Source)
	}

	if result.Finalized {
		t.Error("Expected finalized to be false while pending bounce check")
	}

	if result.NextCheckAt == 0 {
		t.Error("Expected next_check_at to be set")
	}
}

// TestBounceCheckProcess tests the scheduled bounce check processing
func TestBounceCheckProcess_NoBounce(t *testing.T) {
	mockRepo := NewMockRepository()
	mockWebhook := &MockWebhookDispatcher{}

	// Add SMTP account
	smtpAccount := &store.SMTPAccount{
		ID:          "smtp-account-1",
		Host:        "smtp.example.com",
		IMAPHost:    "imap.example.com",
		IMAPPort:    993,
		Username:    "test@example.com",
		Password:    "password",
		IMAPMailbox: "INBOX",
		Active:      true,
	}
	mockRepo.AddSMTPAccount(smtpAccount)

	// Add user with webhook
	testUser := &store.User{
		ID:         "test-user-id",
		WebhookURL: "https://webhook.example.com/notify",
		Active:     true,
	}
	mockRepo.AddUser(testUser)

	// Add a due verification record
	dueRecord := store.VerificationRecord{
		ID:             "ver-123",
		Email:          "test@catchall.com",
		UserID:         testUser.ID,
		Status:         "pending_bounce_check",
		Message:        "probe sent",
		Source:         "smtp-probe",
		ProbeToken:     "token-abc",
		SMTPAccountID:  smtpAccount.ID,
		CheckCount:     0,
		Finalized:      false,
		NextCheckAt:    time.Now().Add(-1 * time.Hour).Unix(),
		FirstCheckedAt: time.Now().Add(-7 * time.Hour).Unix(),
		LastCheckedAt:  time.Now().Add(-7 * time.Hour).Unix(),
		CreatedAt:      time.Now().Add(-7 * time.Hour).Unix(),
		UpdatedAt:      time.Now().Add(-7 * time.Hour).Unix(),
	}
	mockRepo.AddDueCheck(dueRecord)
	mockRepo.UpsertVerification(context.Background(), &dueRecord)

	// Process due checks with no bounce detected
	processedRecord, err := simulateBounceCheck(mockRepo, mockWebhook, &dueRecord, smtpAccount, testUser, false, "")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// After processing with no bounce, record should be finalized
	if !processedRecord.Finalized {
		t.Error("Expected record to be finalized after bounce check")
	}

	if processedRecord.CheckCount != 1 {
		t.Errorf("Expected check_count to be 1, got: %d", processedRecord.CheckCount)
	}

	// Webhook should be called with user's webhook URL
	if len(mockWebhook.SentEvents) != 1 {
		t.Errorf("Expected 1 webhook event, got: %d", len(mockWebhook.SentEvents))
	}

	if mockWebhook.SentEvents[0].WebhookURL != testUser.WebhookURL {
		t.Errorf("Expected webhook URL '%s', got: %s", testUser.WebhookURL, mockWebhook.SentEvents[0].WebhookURL)
	}
}

// TestBounceCheckProcess_BounceDetected tests bounce detection
func TestBounceCheckProcess_BounceDetected(t *testing.T) {
	mockRepo := NewMockRepository()
	mockWebhook := &MockWebhookDispatcher{}

	smtpAccount := &store.SMTPAccount{
		ID:          "smtp-account-1",
		Host:        "smtp.example.com",
		IMAPHost:    "imap.example.com",
		IMAPPort:    993,
		Username:    "test@example.com",
		Password:    "password",
		IMAPMailbox: "INBOX",
		Active:      true,
	}
	mockRepo.AddSMTPAccount(smtpAccount)

	testUser := &store.User{
		ID:         "test-user-id",
		WebhookURL: "https://webhook.example.com/notify",
		Active:     true,
	}
	mockRepo.AddUser(testUser)

	dueRecord := store.VerificationRecord{
		ID:             "ver-456",
		Email:          "bounced@invalid.com",
		UserID:         testUser.ID,
		Status:         "pending_bounce_check",
		Source:         "smtp-probe",
		ProbeToken:     "token-xyz",
		SMTPAccountID:  smtpAccount.ID,
		Finalized:      false,
		NextCheckAt:    time.Now().Add(-1 * time.Hour).Unix(),
		FirstCheckedAt: time.Now().Add(-7 * time.Hour).Unix(),
		LastCheckedAt:  time.Now().Add(-7 * time.Hour).Unix(),
		CreatedAt:      time.Now().Add(-7 * time.Hour).Unix(),
		UpdatedAt:      time.Now().Add(-7 * time.Hour).Unix(),
	}
	mockRepo.AddDueCheck(dueRecord)
	mockRepo.UpsertVerification(context.Background(), &dueRecord)

	// Process with bounce detected
	processedRecord, err := simulateBounceCheck(mockRepo, mockWebhook, &dueRecord, smtpAccount, testUser, true, "Mail delivery failed: user unknown")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// After bounce detected, status should be "bounced"
	if processedRecord.Status != "bounced" {
		t.Errorf("Expected status 'bounced', got: %s", processedRecord.Status)
	}

	if !processedRecord.Finalized {
		t.Error("Expected record to be finalized after bounce detection")
	}

	// Webhook event should be "verify.bounced"
	if len(mockWebhook.SentEvents) != 1 {
		t.Fatalf("Expected 1 webhook event, got: %d", len(mockWebhook.SentEvents))
	}

	if mockWebhook.SentEvents[0].Event != "verify.bounced" {
		t.Errorf("Expected webhook event 'verify.bounced', got: %s", mockWebhook.SentEvents[0].Event)
	}
}

// TestCachedVerification tests that duplicate emails return cached results
func TestCachedVerification(t *testing.T) {
	mockRepo := NewMockRepository()
	mockWebhook := &MockWebhookDispatcher{}

	testUser := &store.User{
		ID:     "test-user-id",
		Active: true,
	}
	mockRepo.AddUser(testUser)

	// Pre-populate a verification record
	existingRecord := &store.VerificationRecord{
		ID:        "existing-ver-id",
		Email:     "existing@example.com",
		UserID:    testUser.ID,
		Status:    "valid",
		Message:   "Email is valid",
		Source:    "direct-smtp-check",
		Finalized: true,
	}
	mockRepo.UpsertVerification(context.Background(), existingRecord)

	// Request verification for same email
	cfg := ServiceConfig{
		FirstBounceDelay:  2 * time.Minute,
		SecondBounceDelay: 6 * time.Hour,
		CheckInterval:     1 * time.Minute,
	}

	result, err := simulateVerificationCached(mockRepo, mockWebhook, cfg, "existing@example.com", testUser)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return cached result
	if !result.Cached {
		t.Error("Expected result to be marked as cached")
	}

	if result.Status != "valid" {
		t.Errorf("Expected cached status 'valid', got: %s", result.Status)
	}

	// No webhook should be sent for cached results
	if len(mockWebhook.SentEvents) != 0 {
		t.Errorf("Expected no webhook events for cached result, got: %d", len(mockWebhook.SentEvents))
	}
}

// Helper functions for simulating verification flows

func simulateVerification(repo *MockRepository, webhook *MockWebhookDispatcher, cfg ServiceConfig, email string, user *store.User, directStatus string) (VerifyResponse, error) {
	now := time.Now().Unix()

	// Check for existing
	existing, _ := repo.GetByEmailAndUser(context.Background(), email, user.ID)
	if existing != nil {
		return responseFromRecord(existing, true), nil
	}

	record := &store.VerificationRecord{
		ID:             "ver-" + email,
		Email:          email,
		UserID:         user.ID,
		Status:         directStatus,
		Message:        "Direct SMTP check result: " + directStatus,
		Source:         "direct-smtp-check",
		FirstCheckedAt: now,
		LastCheckedAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
		Finalized:      true,
	}

	repo.UpsertVerification(context.Background(), record)
	repo.AddEvent(context.Background(), record.ID, "verify.requested", record.Status, record.Message)

	webhookURL := ""
	if user != nil && user.WebhookURL != "" {
		webhookURL = user.WebhookURL
	}
	webhook.SendWithURL(context.Background(), "verify.created", record, webhookURL)

	return responseFromRecord(record, false), nil
}

func simulateVerificationWithProbe(repo *MockRepository, webhook *MockWebhookDispatcher, cfg ServiceConfig, email string, user *store.User, directStatus string) (VerifyResponse, error) {
	now := time.Now().Unix()

	record := &store.VerificationRecord{
		ID:             "ver-" + email,
		Email:          email,
		UserID:         user.ID,
		Status:         directStatus,
		Message:        "Direct check returned: " + directStatus,
		Source:         "direct-smtp-check",
		FirstCheckedAt: now,
		LastCheckedAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Fallback to probe
	if directStatus == "unknown" || directStatus == "error" || directStatus == "greylisted" {
		record.ProbeToken = "token-" + email
		record.Source = "smtp-probe"

		// Simulate acquiring SMTP account
		account, _ := repo.AcquireSMTPAccountForSendByUser(context.Background(), user.ID)
		if account != nil {
			record.SMTPAccountID = account.ID
			record.Status = "pending_bounce_check"
			record.Message = "probe sent via smtp account " + account.ID + "; single bounce check scheduled at 6h"
			record.NextCheckAt = time.Now().Add(cfg.SecondBounceDelay).Unix()
			record.Finalized = false
		} else {
			record.Status = "error"
			record.Message = "fallback probe send failed: no active smtp account"
			record.Finalized = true
		}
	} else {
		record.Finalized = true
	}

	repo.UpsertVerification(context.Background(), record)
	repo.AddEvent(context.Background(), record.ID, "verify.requested", record.Status, record.Message)

	webhookURL := ""
	if user != nil && user.WebhookURL != "" {
		webhookURL = user.WebhookURL
	}
	webhook.SendWithURL(context.Background(), "verify.created", record, webhookURL)

	return responseFromRecord(record, false), nil
}

func simulateBounceCheck(repo *MockRepository, webhook *MockWebhookDispatcher, rec *store.VerificationRecord, account *store.SMTPAccount, user *store.User, bounced bool, bounceReason string) (*store.VerificationRecord, error) {
	now := time.Now().Unix()
	rec.LastCheckedAt = now
	rec.UpdatedAt = now
	rec.CheckCount++
	rec.NextCheckAt = 0
	rec.Finalized = true

	event := "verify.check.no_bounce"
	if bounced {
		rec.Status = "bounced"
		rec.Message = bounceReason
		event = "verify.bounced"
	} else {
		rec.Message = "no bounce detected in single scheduled check; keeping existing status"
	}

	repo.UpsertVerification(context.Background(), rec)
	repo.AddEvent(context.Background(), rec.ID, event, rec.Status, rec.Message)

	webhookURL := ""
	if user != nil && user.WebhookURL != "" {
		webhookURL = user.WebhookURL
	}
	webhook.SendWithURL(context.Background(), event, rec, webhookURL)

	return rec, nil
}

func simulateVerificationCached(repo *MockRepository, webhook *MockWebhookDispatcher, cfg ServiceConfig, email string, user *store.User) (VerifyResponse, error) {
	existing, _ := repo.GetByEmailAndUser(context.Background(), email, user.ID)
	if existing != nil {
		return responseFromRecord(existing, true), nil
	}
	return VerifyResponse{}, nil
}
