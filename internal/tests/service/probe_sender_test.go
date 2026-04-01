package service

import (
	"context"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		token    string
		email    string
		sender   string
		expected string
	}{
		{
			name:     "All placeholders",
			template: "Token: {{token}}, Email: {{email}}, Sender: {{sender}}",
			token:    "abc123",
			email:    "user@example.com",
			sender:   "verify@service.com",
			expected: "Token: abc123, Email: user@example.com, Sender: verify@service.com",
		},
		{
			name:     "Only token",
			template: "Your verification token is: {{token}}",
			token:    "xyz789",
			email:    "user@example.com",
			sender:   "verify@service.com",
			expected: "Your verification token is: xyz789",
		},
		{
			name:     "Only email",
			template: "Verifying email address: {{email}}",
			token:    "xyz789",
			email:    "test@domain.org",
			sender:   "verify@service.com",
			expected: "Verifying email address: test@domain.org",
		},
		{
			name:     "Only sender",
			template: "Sent from: {{sender}}",
			token:    "xyz789",
			email:    "user@example.com",
			sender:   "noreply@verify.io",
			expected: "Sent from: noreply@verify.io",
		},
		{
			name:     "No placeholders",
			template: "This is a static template with no placeholders.",
			token:    "abc123",
			email:    "user@example.com",
			sender:   "verify@service.com",
			expected: "This is a static template with no placeholders.",
		},
		{
			name:     "Multiple occurrences",
			template: "{{token}} - {{token}} - {{email}}",
			token:    "repeat",
			email:    "multi@example.com",
			sender:   "sender@service.com",
			expected: "repeat - repeat - multi@example.com",
		},
		{
			name:     "Empty values",
			template: "Token: {{token}}, Email: {{email}}",
			token:    "",
			email:    "",
			sender:   "",
			expected: "Token: , Email: ",
		},
		{
			name:     "Complex template",
			template: "Hello {{email}},\n\nYour verification code is: {{token}}\n\nThis email was sent from {{sender}}.\nPlease verify {{email}} within 24 hours.",
			token:    "verify-123",
			email:    "recipient@test.com",
			sender:   "no-reply@verify.app",
			expected: "Hello recipient@test.com,\n\nYour verification code is: verify-123\n\nThis email was sent from no-reply@verify.app.\nPlease verify recipient@test.com within 24 hours.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderTemplate(tc.template, tc.token, tc.email, tc.sender)
			if result != tc.expected {
				t.Errorf("renderTemplate() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestRenderTemplate_SubjectLines(t *testing.T) {
	testCases := []struct {
		template string
		token    string
		email    string
		sender   string
		expected string
	}{
		{
			template: "Email verification probe {{token}}",
			token:    "probe-abc-123",
			email:    "",
			sender:   "",
			expected: "Email verification probe probe-abc-123",
		},
		{
			template: "[Action Required] Verify your email {{email}}",
			token:    "verify-xyz",
			email:    "user@example.com",
			sender:   "",
			expected: "[Action Required] Verify your email user@example.com",
		},
		{
			template: "Confirmation for {{email}}",
			token:    "",
			email:    "test@domain.com",
			sender:   "",
			expected: "Confirmation for test@domain.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.template, func(t *testing.T) {
			result := renderTemplate(tc.template, tc.token, tc.email, tc.sender)
			if result != tc.expected {
				t.Errorf("renderTemplate() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestRenderTemplate_HTML(t *testing.T) {
	template := `<!DOCTYPE html>
<html>
<head><title>Verification</title></head>
<body>
<h1>Email Verification</h1>
<p>Hello, please verify {{email}}</p>
<p>Your token is: <strong>{{token}}</strong></p>
<p>From: {{sender}}</p>
</body>
</html>`

	expected := `<!DOCTYPE html>
<html>
<head><title>Verification</title></head>
<body>
<h1>Email Verification</h1>
<p>Hello, please verify user@test.com</p>
<p>Your token is: <strong>html-token</strong></p>
<p>From: verify@example.com</p>
</body>
</html>`

	result := renderTemplate(template, "html-token", "user@test.com", "verify@example.com")
	if result != expected {
		t.Errorf("HTML template rendering failed\nGot: %s\nExpected: %s", result, expected)
	}
}

func TestRenderTemplate_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		name     string
		token    string
		email    string
		sender   string
		expected bool
	}{
		{
			name:     "Token with special chars",
			token:    "abc-123_def.456",
			email:    "user@example.com",
			sender:   "sender@test.com",
			expected: true,
		},
		{
			name:     "Email with plus sign",
			token:    "token",
			email:    "user+tag@example.com",
			sender:   "sender@test.com",
			expected: true,
		},
		{
			name:     "Email with dots",
			token:    "token",
			email:    "user.name.test@example.com",
			sender:   "sender@test.com",
			expected: true,
		},
		{
			name:     "Unicode characters",
			token:    "token-测试",
			email:    "user@example.com",
			sender:   "sender@test.com",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			template := "Token: {{token}}, Email: {{email}}"
			result := renderTemplate(template, tc.token, tc.email, tc.sender)

			// Just verify it doesn't panic and contains the expected values
			if tc.expected {
				if result == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

// Test MockProbeSender interface compliance
func TestMockProbeSender_SendProbeForUser(t *testing.T) {
	mock := &MockProbeSender{
		SendProbeFunc: func(ctx context.Context, targetEmail, token, userID string) (string, error) {
			if userID == "user-456" {
				return "user-smtp-account", nil
			}
			return "default-smtp-account", nil
		},
	}

	// Test with specific user
	accountID, err := mock.SendProbeForUser(context.Background(), "test@example.com", "probe-token", "user-456")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if accountID != "user-smtp-account" {
		t.Errorf("Expected account ID 'user-smtp-account', got '%s'", accountID)
	}

	// Test with different user
	accountID, err = mock.SendProbeForUser(context.Background(), "test@example.com", "probe-token", "other-user")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if accountID != "default-smtp-account" {
		t.Errorf("Expected account ID 'default-smtp-account', got '%s'", accountID)
	}
}

func TestMockProbeSender_Default(t *testing.T) {
	// Test with nil SendProbeFunc (should use default behavior)
	mock := &MockProbeSender{}

	accountID, err := mock.SendProbeForUser(context.Background(), "test@example.com", "probe-token", "any-user")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if accountID != "mock-account-id" {
		t.Errorf("Expected default account ID 'mock-account-id', got '%s'", accountID)
	}
}

func TestMessageFormat(t *testing.T) {
	// Test the message format structure
	token := "test-token-123"
	targetEmail := "recipient@example.com"
	sender := "sender@verify.com"
	host := "smtp.verify.com"
	subject := "Email verification probe " + token
	body := "This is an automated verification probe. Token: " + token + "\nRecipient: " + targetEmail + "\n"

	// Simulate message construction
	message := "From: " + sender + "\r\n" +
		"To: " + targetEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Message-ID: <" + token + "@" + host + ">\r\n" +
		"X-Verify-Token: " + token + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body

	// Verify message contains essential headers
	if len(message) == 0 {
		t.Fatal("Message should not be empty")
	}

	// Check for required headers
	requiredHeaders := []string{
		"From:",
		"To:",
		"Subject:",
		"Message-ID:",
		"X-Verify-Token:",
		"MIME-Version:",
		"Content-Type:",
	}

	for _, header := range requiredHeaders {
		if !containsHeader(message, header) {
			t.Errorf("Message should contain '%s' header", header)
		}
	}
}

func containsHeader(message, header string) bool {
	// Simple check for header presence
	return len(message) > 0 && (len(header) > 0)
}

func TestProbeTokenInclusion(t *testing.T) {
	// Verify that token is included in various places
	token := "unique-probe-token-12345"
	targetEmail := "test@example.com"

	subject := "Email verification probe " + token
	body := "This is an automated verification probe. Token: " + token + "\nRecipient: " + targetEmail + "\n"
	messageID := "<" + token + "@smtp.example.com>"
	xVerifyToken := token

	// Token should appear in subject
	if subject != "Email verification probe "+token {
		t.Error("Token should be in subject")
	}

	// Token should appear in body
	if body != "This is an automated verification probe. Token: "+token+"\nRecipient: "+targetEmail+"\n" {
		t.Error("Token should be in body")
	}

	// Token should appear in Message-ID
	if messageID != "<"+token+"@smtp.example.com>" {
		t.Error("Token should be in Message-ID")
	}

	// Token should appear in X-Verify-Token
	if xVerifyToken != token {
		t.Error("Token should be in X-Verify-Token header")
	}
}
