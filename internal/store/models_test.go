package store

import (
	"encoding/json"
	"testing"
)

func TestUser_JSONSerialization(t *testing.T) {
	user := User{
		ID:         "user-123",
		Name:       "Test User",
		Email:      "test@example.com",
		APIKey:     "evk_test123",
		WebhookURL: "https://webhook.example.com",
		Active:     true,
		CreatedAt:  1704067200,
		UpdatedAt:  1704067200,
	}

	jsonData, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	var unmarshaled User
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if unmarshaled.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, unmarshaled.ID)
	}

	if unmarshaled.Name != user.Name {
		t.Errorf("Expected Name %s, got %s", user.Name, unmarshaled.Name)
	}

	if unmarshaled.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, unmarshaled.Email)
	}

	if unmarshaled.Active != user.Active {
		t.Errorf("Expected Active %v, got %v", user.Active, unmarshaled.Active)
	}
}

func TestVerificationRecord_Structure(t *testing.T) {
	rec := VerificationRecord{
		ID:             "ver-123",
		Email:          "test@example.com",
		Status:         "valid",
		Message:        "250 Accepted",
		Source:         "direct_smtp",
		ProbeToken:     "probe-token-xyz",
		SMTPAccountID:  "smtp-1",
		UserID:         "user-456",
		CheckCount:     3,
		Finalized:      true,
		FirstCheckedAt: 1704067200,
		LastCheckedAt:  1704153600,
		NextCheckAt:    0,
		CreatedAt:      1704067200,
		UpdatedAt:      1704153600,
	}

	if rec.ID != "ver-123" {
		t.Errorf("Expected ID 'ver-123', got %s", rec.ID)
	}

	if rec.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got %s", rec.Email)
	}

	if rec.Status != "valid" {
		t.Errorf("Expected Status 'valid', got %s", rec.Status)
	}

	if rec.CheckCount != 3 {
		t.Errorf("Expected CheckCount 3, got %d", rec.CheckCount)
	}

	if !rec.Finalized {
		t.Error("Expected Finalized to be true")
	}
}

func TestSMTPAccount_JSONSerialization(t *testing.T) {
	account := SMTPAccount{
		ID:          "smtp-123",
		UserID:      "user-456",
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "smtp-user",
		Password:    "secret-password",
		Sender:      "noreply@example.com",
		IMAPHost:    "imap.example.com",
		IMAPPort:    993,
		IMAPMailbox: "INBOX",
		DailyLimit:  100,
		SentToday:   25,
		ResetDate:   "2024-01-01",
		Active:      true,
		CreatedAt:   1704067200,
		UpdatedAt:   1704067200,
	}

	jsonData, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("Failed to marshal account: %v", err)
	}

	// Password should be hidden in JSON (json:"-")
	if string(jsonData) == "" {
		t.Error("JSON should not be empty")
	}

	jsonStr := string(jsonData)
	if jsonStr != "" && contains(jsonStr, "secret-password") {
		t.Error("Password should not be visible in JSON output")
	}

	var unmarshaled SMTPAccount
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal account: %v", err)
	}

	if unmarshaled.Host != account.Host {
		t.Errorf("Expected Host %s, got %s", account.Host, unmarshaled.Host)
	}

	if unmarshaled.Port != account.Port {
		t.Errorf("Expected Port %d, got %d", account.Port, unmarshaled.Port)
	}

	// Password should be empty after unmarshaling (because it's not serialized)
	if unmarshaled.Password != "" {
		t.Error("Password should not be unmarshaled from JSON")
	}
}

func TestEmailTemplate_Structure(t *testing.T) {
	template := EmailTemplate{
		ID:              "tpl-123",
		UserID:          "user-456",
		Name:            "default",
		SubjectTemplate: "Verification for {{email}}",
		BodyTemplate:    "Your token is: {{token}}",
		Active:          true,
		CreatedAt:       1704067200,
		UpdatedAt:       1704067200,
	}

	if template.ID != "tpl-123" {
		t.Errorf("Expected ID 'tpl-123', got %s", template.ID)
	}

	if template.Name != "default" {
		t.Errorf("Expected Name 'default', got %s", template.Name)
	}

	if template.SubjectTemplate != "Verification for {{email}}" {
		t.Errorf("Expected SubjectTemplate 'Verification for {{email}}', got %s", template.SubjectTemplate)
	}

	if !template.Active {
		t.Error("Expected Active to be true")
	}
}

func TestUserInput_Structure(t *testing.T) {
	input := UserInput{
		ID:         "user-input-1",
		Name:       "New User",
		Email:      "new@example.com",
		APIKey:     "evk_newkey",
		WebhookURL: "https://new-webhook.com",
		Active:     true,
	}

	if input.ID != "user-input-1" {
		t.Errorf("Expected ID 'user-input-1', got %s", input.ID)
	}

	if input.Name != "New User" {
		t.Errorf("Expected Name 'New User', got %s", input.Name)
	}
}

func TestSMTPAccountInput_Structure(t *testing.T) {
	input := SMTPAccountInput{
		ID:          "smtp-input-1",
		UserID:      "user-123",
		Host:        "smtp.test.com",
		Port:        465,
		Username:    "test-user",
		Password:    "test-password",
		Sender:      "test@test.com",
		IMAPHost:    "imap.test.com",
		IMAPPort:    993,
		IMAPMailbox: "INBOX",
		DailyLimit:  50,
		Active:      true,
	}

	if input.Host != "smtp.test.com" {
		t.Errorf("Expected Host 'smtp.test.com', got %s", input.Host)
	}

	if input.Port != 465 {
		t.Errorf("Expected Port 465, got %d", input.Port)
	}

	if input.DailyLimit != 50 {
		t.Errorf("Expected DailyLimit 50, got %d", input.DailyLimit)
	}
}

func TestEmailTemplateInput_Structure(t *testing.T) {
	input := EmailTemplateInput{
		ID:              "tpl-input-1",
		UserID:          "user-123",
		Name:            "custom",
		SubjectTemplate: "Custom subject for {{email}}",
		BodyTemplate:    "Custom body with {{token}}",
		Active:          true,
	}

	if input.Name != "custom" {
		t.Errorf("Expected Name 'custom', got %s", input.Name)
	}

	if input.SubjectTemplate != "Custom subject for {{email}}" {
		t.Errorf("Expected SubjectTemplate, got %s", input.SubjectTemplate)
	}

	if !input.Active {
		t.Error("Expected Active to be true")
	}
}

func TestVerificationRecord_StatusValues(t *testing.T) {
	statuses := []string{"valid", "invalid", "pending", "unknown", "disposable", "greylisted", "error"}

	for _, status := range statuses {
		rec := VerificationRecord{
			ID:     "ver-" + status,
			Email:  "test@example.com",
			Status: status,
		}

		if rec.Status != status {
			t.Errorf("Expected Status %s, got %s", status, rec.Status)
		}
	}
}

func TestVerificationRecord_SourceValues(t *testing.T) {
	sources := []string{"direct_smtp", "probe", "cached"}

	for _, source := range sources {
		rec := VerificationRecord{
			ID:     "ver-src",
			Email:  "test@example.com",
			Source: source,
		}

		if rec.Source != source {
			t.Errorf("Expected Source %s, got %s", source, rec.Source)
		}
	}
}

func TestUser_ActiveFlag(t *testing.T) {
	activeUser := User{ID: "1", Active: true}
	inactiveUser := User{ID: "2", Active: false}

	if !activeUser.Active {
		t.Error("Expected active user to be active")
	}

	if inactiveUser.Active {
		t.Error("Expected inactive user to be inactive")
	}
}

func TestSMTPAccount_DailyLimitTracking(t *testing.T) {
	account := SMTPAccount{
		ID:         "smtp-1",
		DailyLimit: 100,
		SentToday:  0,
	}

	// Simulate sending emails
	for i := 0; i < 50; i++ {
		account.SentToday++
	}

	if account.SentToday != 50 {
		t.Errorf("Expected SentToday 50, got %d", account.SentToday)
	}

	// Check if still under limit
	if account.SentToday >= account.DailyLimit {
		t.Error("Should still be under daily limit")
	}

	// Send more to reach limit
	for i := 0; i < 50; i++ {
		account.SentToday++
	}

	if account.SentToday < account.DailyLimit {
		t.Error("Should have reached daily limit")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
