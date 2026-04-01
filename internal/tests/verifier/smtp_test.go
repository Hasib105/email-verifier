package verifier

import (
	"testing"
)

func TestIsValidEmailSyntax(t *testing.T) {
	testCases := []struct {
		email    string
		expected bool
	}{
		// Valid emails
		{"test@example.com", true},
		{"user.name@domain.com", true},
		{"user+tag@example.com", true},
		{"user123@example.org", true},
		{"a@b.co", true},
		{"user_name@example.com", true},
		{"user-name@example.com", true},
		{"test@subdomain.example.com", true},

		// Invalid emails
		{"", false},
		{"plaintext", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
		{"user@domain", false},
		{"user @example.com", false},
		{"user@ example.com", false},
		{"user@example .com", false},
		{"user@@example.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			result := isValidEmailSyntax(tc.email)
			if result != tc.expected {
				t.Errorf("isValidEmailSyntax(%q) = %v, expected %v", tc.email, result, tc.expected)
			}
		})
	}
}

func TestIsDisposableDomain(t *testing.T) {
	testCases := []struct {
		domain   string
		expected bool
	}{
		// Known disposable domains
		{"mailinator.com", true},
		{"yopmail.com", true},
		{"10minutemail.com", true},

		// Regular domains
		{"gmail.com", false},
		{"yahoo.com", false},
		{"example.com", false},
		{"company.org", false},
	}

	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			result := isDisposableDomain(tc.domain)
			if result != tc.expected {
				t.Errorf("isDisposableDomain(%q) = %v, expected %v", tc.domain, result, tc.expected)
			}
		})
	}
}

func TestEmailVerifier_New(t *testing.T) {
	verifier := New("sender@example.com", "example.com", "127.0.0.1:9050", 5, 30)

	if verifier.FromEmail != "sender@example.com" {
		t.Errorf("Expected FromEmail 'sender@example.com', got %s", verifier.FromEmail)
	}

	if verifier.EHLODomain != "example.com" {
		t.Errorf("Expected EHLODomain 'example.com', got %s", verifier.EHLODomain)
	}

	if verifier.ProxyAddr != "127.0.0.1:9050" {
		t.Errorf("Expected ProxyAddr '127.0.0.1:9050', got %s", verifier.ProxyAddr)
	}

	if verifier.Timeout != 30 {
		t.Errorf("Expected Timeout 30, got %v", verifier.Timeout)
	}

	// Check semaphore capacity
	if cap(verifier.Semaphore) != 5 {
		t.Errorf("Expected Semaphore capacity 5, got %d", cap(verifier.Semaphore))
	}
}

func TestVerifyResult_Structure(t *testing.T) {
	result := VerifyResult{
		Status:  "valid",
		Message: "250 Accepted",
		Email:   "test@example.com",
	}

	if result.Status != "valid" {
		t.Errorf("Expected Status 'valid', got %s", result.Status)
	}

	if result.Message != "250 Accepted" {
		t.Errorf("Expected Message '250 Accepted', got %s", result.Message)
	}

	if result.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got %s", result.Email)
	}
}

// Test that Verify correctly handles invalid syntax without network calls
func TestEmailVerifier_Verify_InvalidSyntax(t *testing.T) {
	verifier := New("sender@example.com", "example.com", "", 1, 30)

	testCases := []struct {
		email          string
		expectedStatus string
	}{
		{"not-an-email", "invalid"},
		{"@example.com", "invalid"},
		{"user@", "invalid"},
		{"", "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			result := verifier.Verify(tc.email)
			if result.Status != tc.expectedStatus {
				t.Errorf("Verify(%q) status = %s, expected %s", tc.email, result.Status, tc.expectedStatus)
			}
			if result.Message != "invalid syntax" {
				t.Errorf("Verify(%q) message = %s, expected 'invalid syntax'", tc.email, result.Message)
			}
		})
	}
}

// Test that Verify correctly handles disposable domains without network calls
func TestEmailVerifier_Verify_DisposableDomain(t *testing.T) {
	verifier := New("sender@example.com", "example.com", "", 1, 30)

	testCases := []string{
		"user@mailinator.com",
		"test@yopmail.com",
		"temp@10minutemail.com",
	}

	for _, email := range testCases {
		t.Run(email, func(t *testing.T) {
			result := verifier.Verify(email)
			if result.Status != "disposable" {
				t.Errorf("Verify(%q) status = %s, expected 'disposable'", email, result.Status)
			}
			if result.Message != "disposable domain detected" {
				t.Errorf("Verify(%q) message = %s, expected 'disposable domain detected'", email, result.Message)
			}
		})
	}
}

// Test email normalization (lowercase, trim)
func TestEmailVerifier_Verify_Normalization(t *testing.T) {
	verifier := New("sender@example.com", "example.com", "", 1, 30)

	// Test that email is normalized before validation
	// Using a disposable domain to avoid network calls
	result := verifier.Verify("  TEST@MAILINATOR.COM  ")

	if result.Email != "test@mailinator.com" {
		t.Errorf("Expected normalized email 'test@mailinator.com', got %s", result.Email)
	}

	if result.Status != "disposable" {
		t.Errorf("Expected status 'disposable', got %s", result.Status)
	}
}

// Test concurrent access with semaphore
func TestEmailVerifier_Concurrency(t *testing.T) {
	verifier := New("sender@example.com", "example.com", "", 2, 30)

	done := make(chan bool, 5)

	// Launch 5 concurrent verifications (semaphore allows 2)
	for i := 0; i < 5; i++ {
		go func() {
			result := verifier.Verify("test@mailinator.com")
			if result.Status != "disposable" {
				t.Errorf("Expected 'disposable', got %s", result.Status)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// Test different email formats
func TestEmailFormats(t *testing.T) {
	testCases := []struct {
		name     string
		email    string
		expected bool
	}{
		{"simple email", "user@example.com", true},
		{"with dots", "user.name@example.com", true},
		{"with plus", "user+tag@example.com", true},
		{"with numbers", "user123@example.com", true},
		{"subdomain", "user@mail.example.com", true},
		{"short TLD", "user@example.io", true},
		{"long TLD", "user@example.technology", true},
		{"underscore", "user_name@example.com", true},
		{"hyphen", "user-name@example.com", true},
		{"percent", "user%name@example.com", true},

		// Invalid formats
		{"empty", "", false},
		{"no at", "userexample.com", false},
		{"no domain", "user@", false},
		{"no user", "@example.com", false},
		{"double at", "user@@example.com", false},
		{"space in local", "user name@example.com", false},
		{"space in domain", "user@example .com", false},
		{"no TLD", "user@example", false},
		{"single char TLD", "user@example.a", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidEmailSyntax(tc.email)
			if result != tc.expected {
				t.Errorf("isValidEmailSyntax(%q) = %v, expected %v", tc.email, result, tc.expected)
			}
		})
	}
}
