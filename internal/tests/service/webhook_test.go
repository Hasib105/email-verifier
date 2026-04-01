package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"email-verifier-api/internal/store"
)

func TestNewHTTPWebhookDispatcher(t *testing.T) {
	dispatcher := NewHTTPWebhookDispatcher("https://webhook.example.com", 10*time.Second)

	if dispatcher.defaultURL != "https://webhook.example.com" {
		t.Errorf("Expected defaultURL 'https://webhook.example.com', got %s", dispatcher.defaultURL)
	}

	if dispatcher.client == nil {
		t.Fatal("Expected client to be initialized")
	}

	if dispatcher.client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", dispatcher.client.Timeout)
	}
}

func TestHTTPWebhookDispatcher_Send_Success(t *testing.T) {
	// Create a test server to receive the webhook
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse body
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewHTTPWebhookDispatcher(server.URL, 5*time.Second)

	rec := &store.VerificationRecord{
		ID:            "ver-123",
		Email:         "test@example.com",
		Status:        "valid",
		Message:       "250 Accepted",
		Source:        "direct_smtp",
		UserID:        "user-456",
		CheckCount:    1,
		Finalized:     true,
		LastCheckedAt: time.Now().Unix(),
	}

	err := dispatcher.Send(context.Background(), "verification.completed", rec)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify payload
	if receivedPayload["event"] != "verification.completed" {
		t.Errorf("Expected event 'verification.completed', got %v", receivedPayload["event"])
	}

	if receivedPayload["email"] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %v", receivedPayload["email"])
	}

	if receivedPayload["status"] != "valid" {
		t.Errorf("Expected status 'valid', got %v", receivedPayload["status"])
	}

	if receivedPayload["user_id"] != "user-456" {
		t.Errorf("Expected user_id 'user-456', got %v", receivedPayload["user_id"])
	}
}

func TestHTTPWebhookDispatcher_SendWithURL_OverridesDefault(t *testing.T) {
	// Default server that should NOT receive the request
	defaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Default server should not receive request when custom URL is provided")
		w.WriteHeader(http.StatusOK)
	}))
	defer defaultServer.Close()

	// Custom server that SHOULD receive the request
	customServerCalled := false
	customServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customServerCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer customServer.Close()

	dispatcher := NewHTTPWebhookDispatcher(defaultServer.URL, 5*time.Second)

	rec := &store.VerificationRecord{
		ID:     "ver-123",
		Email:  "test@example.com",
		Status: "valid",
	}

	err := dispatcher.SendWithURL(context.Background(), "verification.completed", rec, customServer.URL)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !customServerCalled {
		t.Error("Expected custom server to be called")
	}
}

func TestHTTPWebhookDispatcher_Send_NoURLConfigured(t *testing.T) {
	dispatcher := NewHTTPWebhookDispatcher("", 5*time.Second)

	rec := &store.VerificationRecord{
		ID:     "ver-123",
		Email:  "test@example.com",
		Status: "valid",
	}

	// Should not error when no URL is configured
	err := dispatcher.Send(context.Background(), "verification.completed", rec)
	if err != nil {
		t.Fatalf("Expected no error for empty URL, got: %v", err)
	}
}

func TestHTTPWebhookDispatcher_Send_NilRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Server should not be called for nil record")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewHTTPWebhookDispatcher(server.URL, 5*time.Second)

	// Should not error or make request for nil record
	err := dispatcher.Send(context.Background(), "verification.completed", nil)
	if err != nil {
		t.Fatalf("Expected no error for nil record, got: %v", err)
	}
}

func TestHTTPWebhookDispatcher_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dispatcher := NewHTTPWebhookDispatcher(server.URL, 5*time.Second)

	rec := &store.VerificationRecord{
		ID:     "ver-123",
		Email:  "test@example.com",
		Status: "valid",
	}

	err := dispatcher.Send(context.Background(), "verification.completed", rec)
	if err == nil {
		t.Fatal("Expected error for server error response")
	}

	// Error message should contain status code
	if err.Error() != "webhook returned status 500" {
		t.Errorf("Expected error message about status 500, got: %v", err)
	}
}

func TestHTTPWebhookDispatcher_Send_ServerBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	dispatcher := NewHTTPWebhookDispatcher(server.URL, 5*time.Second)

	rec := &store.VerificationRecord{
		ID:     "ver-123",
		Email:  "test@example.com",
		Status: "valid",
	}

	err := dispatcher.Send(context.Background(), "verification.completed", rec)
	if err == nil {
		t.Fatal("Expected error for bad request response")
	}
}

func TestHTTPWebhookDispatcher_Send_InvalidURL(t *testing.T) {
	dispatcher := NewHTTPWebhookDispatcher("http://invalid-url-that-wont-resolve.local:9999", 1*time.Second)

	rec := &store.VerificationRecord{
		ID:     "ver-123",
		Email:  "test@example.com",
		Status: "valid",
	}

	err := dispatcher.Send(context.Background(), "verification.completed", rec)
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}

func TestHTTPWebhookDispatcher_PayloadFields(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewHTTPWebhookDispatcher(server.URL, 5*time.Second)

	checkedAt := time.Now().Unix()
	rec := &store.VerificationRecord{
		ID:            "ver-abc-123",
		Email:         "user@domain.com",
		Status:        "pending",
		Message:       "probe sent",
		Source:        "probe",
		UserID:        "user-xyz",
		CheckCount:    3,
		Finalized:     false,
		LastCheckedAt: checkedAt,
	}

	err := dispatcher.Send(context.Background(), "verification.probed", rec)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify all payload fields
	expectedFields := map[string]interface{}{
		"event":       "verification.probed",
		"id":          "ver-abc-123",
		"email":       "user@domain.com",
		"status":      "pending",
		"message":     "probe sent",
		"source":      "probe",
		"user_id":     "user-xyz",
		"check_count": float64(3), // JSON numbers are float64
		"finalized":   false,
		"checked_at":  float64(checkedAt),
	}

	for key, expected := range expectedFields {
		if receivedPayload[key] != expected {
			t.Errorf("Payload[%s] = %v, expected %v", key, receivedPayload[key], expected)
		}
	}
}

func TestHTTPWebhookDispatcher_ContextCancellation(t *testing.T) {
	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewHTTPWebhookDispatcher(server.URL, 10*time.Second)

	rec := &store.VerificationRecord{
		ID:     "ver-123",
		Email:  "test@example.com",
		Status: "valid",
	}

	// Create a context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := dispatcher.SendWithURL(ctx, "verification.completed", rec, server.URL)
	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}
}

func TestHTTPWebhookDispatcher_DifferentEvents(t *testing.T) {
	events := []string{
		"verification.started",
		"verification.completed",
		"verification.failed",
		"verification.probed",
		"verification.bounced",
	}

	for _, event := range events {
		t.Run(event, func(t *testing.T) {
			var receivedEvent string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]interface{}
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &payload)
				receivedEvent = payload["event"].(string)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			dispatcher := NewHTTPWebhookDispatcher(server.URL, 5*time.Second)

			rec := &store.VerificationRecord{
				ID:     "ver-123",
				Email:  "test@example.com",
				Status: "valid",
			}

			err := dispatcher.Send(context.Background(), event, rec)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if receivedEvent != event {
				t.Errorf("Expected event '%s', got '%s'", event, receivedEvent)
			}
		})
	}
}

// Test MockWebhookDispatcher for verification service tests
func TestMockWebhookDispatcher_SendCalled(t *testing.T) {
	mock := &MockWebhookDispatcher{}

	rec := &store.VerificationRecord{
		ID:    "ver-123",
		Email: "test@example.com",
	}

	err := mock.Send(context.Background(), "test.event", rec)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(mock.SentEvents) != 1 {
		t.Errorf("Expected 1 sent event, got %d", len(mock.SentEvents))
	}

	if mock.SentEvents[0].Event != "test.event" {
		t.Errorf("Expected event 'test.event', got '%s'", mock.SentEvents[0].Event)
	}
}

func TestMockWebhookDispatcher_SendWithURLCalled(t *testing.T) {
	mock := &MockWebhookDispatcher{}

	rec := &store.VerificationRecord{
		ID:    "ver-123",
		Email: "test@example.com",
	}

	err := mock.SendWithURL(context.Background(), "test.event", rec, "https://custom.webhook.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(mock.SentEvents) != 1 {
		t.Errorf("Expected 1 sent event, got %d", len(mock.SentEvents))
	}

	if mock.SentEvents[0].WebhookURL != "https://custom.webhook.com" {
		t.Errorf("Expected URL 'https://custom.webhook.com', got '%s'", mock.SentEvents[0].WebhookURL)
	}
}
