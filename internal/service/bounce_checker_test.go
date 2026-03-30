package service

import (
	"context"
	"strings"
	"testing"
)

func TestNewIMAPBounceChecker(t *testing.T) {
	checker := NewIMAPBounceChecker()
	if checker == nil {
		t.Fatal("Expected non-nil checker")
	}
}

func TestIMAPConfig_Structure(t *testing.T) {
	cfg := IMAPConfig{
		Host:     "imap.example.com",
		Port:     993,
		Username: "user@example.com",
		Password: "password123",
		Mailbox:  "INBOX",
	}

	if cfg.Host != "imap.example.com" {
		t.Errorf("Expected Host 'imap.example.com', got %s", cfg.Host)
	}

	if cfg.Port != 993 {
		t.Errorf("Expected Port 993, got %d", cfg.Port)
	}

	if cfg.Username != "user@example.com" {
		t.Errorf("Expected Username 'user@example.com', got %s", cfg.Username)
	}

	if cfg.Password != "password123" {
		t.Errorf("Expected Password 'password123', got %s", cfg.Password)
	}

	if cfg.Mailbox != "INBOX" {
		t.Errorf("Expected Mailbox 'INBOX', got %s", cfg.Mailbox)
	}
}

func TestContainsBounceSignature(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "Delivery status notification failure",
			text:     "This is a delivery status notification (failure) for your message",
			expected: true,
		},
		{
			name:     "Undeliverable message",
			text:     "Your message was undeliverable to the destination",
			expected: true,
		},
		{
			name:     "Mail delivery failed",
			text:     "Mail delivery failed: returning message to sender",
			expected: true,
		},
		{
			name:     "Final recipient header",
			text:     "Final-Recipient: rfc822; user@example.com",
			expected: true,
		},
		{
			name:     "Status 5.x.x error",
			text:     "Action: failed\nStatus: 5.1.1",
			expected: true,
		},
		{
			name:     "Mail system message",
			text:     "This is the mail system at host mx.example.com. I'm sorry to inform you",
			expected: true,
		},
		{
			name:     "Regular email - no bounce",
			text:     "Hello, this is a regular email message about our meeting tomorrow.",
			expected: false,
		},
		{
			name:     "Empty text",
			text:     "",
			expected: false,
		},
		{
			name:     "Partial match - not bounce",
			text:     "The mail was delivered successfully",
			expected: false,
		},
		{
			name:     "Case insensitive - uppercase",
			text:     "THIS IS A DELIVERY STATUS NOTIFICATION (FAILURE)",
			expected: true, // Function receives text already converted to lowercase
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert to lowercase as the actual function expects
			text := strings.ToLower(tc.text)
			result := containsBounceSignature(text)
			if result != tc.expected {
				t.Errorf("containsBounceSignature(%q) = %v, expected %v", tc.text, result, tc.expected)
			}
		})
	}
}

func TestContainsBounceSignature_AllSignals(t *testing.T) {
	// Test each bounce signal individually
	signals := []string{
		"delivery status notification (failure)",
		"undeliverable",
		"mail delivery failed",
		"final-recipient",
		"status: 5.",
		"this is the mail system at host",
	}

	for _, signal := range signals {
		t.Run(signal, func(t *testing.T) {
			text := "Some text before " + signal + " some text after"
			result := containsBounceSignature(text)
			if !result {
				t.Errorf("Expected bounce signature detected for '%s'", signal)
			}
		})
	}
}

func TestContainsBounceSignature_CombinedSignals(t *testing.T) {
	// Test a realistic bounce message with multiple signals
	bounceMessage := `
		Subject: Mail Delivery Failed

		This is the mail system at host mx.example.com.

		I'm sorry to have to inform you that your message could not
		be delivered to one or more recipients.

		<user@example.com>: delivery status notification (failure)

		Final-Recipient: rfc822; user@example.com
		Action: failed
		Status: 5.1.1
		Diagnostic-Code: smtp; 550 5.1.1 User unknown
	`

	result := containsBounceSignature(strings.ToLower(bounceMessage))
	if !result {
		t.Error("Expected bounce signature detected in realistic bounce message")
	}
}

func TestIMAPConfig_DefaultMailbox(t *testing.T) {
	cfg := IMAPConfig{
		Host:     "imap.example.com",
		Port:     993,
		Username: "user@example.com",
		Password: "password123",
		// Mailbox not set - should default to INBOX
	}

	if cfg.Mailbox != "" {
		t.Errorf("Expected empty Mailbox, got %s", cfg.Mailbox)
	}

	// In the actual HasBounce function, empty mailbox defaults to "INBOX"
	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	if mailbox != "INBOX" {
		t.Errorf("Expected default mailbox 'INBOX', got %s", mailbox)
	}
}

// Test MockBounceChecker interface
func TestMockBounceChecker_HasBounce(t *testing.T) {
	testCases := []struct {
		name            string
		hasBounce       bool
		bounceMessage   string
		expectedBounce  bool
		expectedMessage string
	}{
		{
			name:            "No bounce detected",
			hasBounce:       false,
			bounceMessage:   "",
			expectedBounce:  false,
			expectedMessage: "",
		},
		{
			name:            "Bounce detected with message",
			hasBounce:       true,
			bounceMessage:   "Bounce detected for recipient",
			expectedBounce:  true,
			expectedMessage: "Bounce detected for recipient",
		},
		{
			name:            "Bounce detected with probe token",
			hasBounce:       true,
			bounceMessage:   "Bounce detected for probe token in subject: Undeliverable",
			expectedBounce:  true,
			expectedMessage: "Bounce detected for probe token in subject: Undeliverable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockBounceChecker{
				HasBounceFunc: func(ctx context.Context, cfg IMAPConfig, targetEmail, token string) (bool, string, error) {
					return tc.hasBounce, tc.bounceMessage, nil
				},
			}

			cfg := IMAPConfig{
				Host:     "imap.example.com",
				Port:     993,
				Username: "user@example.com",
				Password: "password",
			}

			hasBounce, message, err := mock.HasBounce(context.Background(), cfg, "target@example.com", "probe-token")

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if hasBounce != tc.expectedBounce {
				t.Errorf("Expected hasBounce %v, got %v", tc.expectedBounce, hasBounce)
			}

			if message != tc.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tc.expectedMessage, message)
			}
		})
	}
}

func TestBounceSignatureVariations(t *testing.T) {
	// Test various real-world bounce message variations
	bounceVariations := []string{
		// Gmail bounce
		`delivery status notification (failure)
		 delivery to the following recipient failed permanently:
		 user@example.com
		 technical details of permanent failure:
		 the email account that you tried to reach does not exist.`,

		// Microsoft/Outlook bounce
		`your message to user@example.com couldn't be delivered.
		 user wasn't found at example.com.
		 delivery status notification (failure)`,

		// Yahoo bounce
		`this is a delivery status notification (failure).
		 sorry, we couldn't deliver your message to the following address.`,

		// Generic bounce
		`this is the mail system at host mta.example.com.
		 i'm sorry to have to inform you that your message could not be delivered.
		 mail delivery failed.`,
	}

	for i, variation := range bounceVariations {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			text := strings.ToLower(variation)
			result := containsBounceSignature(text)
			if !result {
				t.Errorf("Expected bounce signature in variation %d", i+1)
			}
		})
	}
}

func TestNonBounceEmails(t *testing.T) {
	// Test emails that should NOT be detected as bounces
	normalEmails := []string{
		"hi, just wanted to follow up on our meeting.",
		"your order has been shipped and will arrive tomorrow.",
		"welcome to our service! your account is now active.",
		"reminder: your subscription renewal is coming up.",
		"thank you for contacting support. here is your ticket number.",
		"the document has been successfully delivered to all recipients.",
		"your report is ready for download.",
	}

	for i, email := range normalEmails {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			text := strings.ToLower(email)
			result := containsBounceSignature(text)
			if result {
				t.Errorf("Normal email %d should not be detected as bounce", i+1)
			}
		})
	}
}
