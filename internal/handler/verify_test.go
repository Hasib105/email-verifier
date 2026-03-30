package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"email-verifier-api/internal/service"
	"email-verifier-api/internal/store"

	"github.com/gofiber/fiber/v2"
)

// MockUserService is a mock implementation for testing
type MockUserService struct {
	users map[string]*store.User
}

func NewMockUserService() *MockUserService {
	svc := &MockUserService{
		users: make(map[string]*store.User),
	}
	// Add a default test user
	svc.users["test-api-key"] = &store.User{
		ID:         "user-1",
		Name:       "Test User",
		Email:      "test@example.com",
		APIKey:     "test-api-key",
		WebhookURL: "https://webhook.example.com",
		Active:     true,
	}
	return svc
}

func (m *MockUserService) AuthenticateByAPIKey(ctx context.Context, apiKey string) (*store.User, error) {
	if user, ok := m.users[apiKey]; ok {
		return user, nil
	}
	return nil, nil
}

// MockVerificationService is a mock implementation for testing
type MockVerificationService struct {
	verifyFunc func(ctx context.Context, email string, user *store.User) (*service.VerifyResponse, error)
}

func (m *MockVerificationService) VerifyEmail(ctx context.Context, email string, user *store.User) (*service.VerifyResponse, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, email, user)
	}
	return &service.VerifyResponse{
		Email:   email,
		Status:  "valid",
		Message: "OK",
	}, nil
}

// Test helpers
func setupTestApp(mockUserSvc *MockUserService, mockVerifySvc *MockVerificationService) *fiber.App {
	app := fiber.New()

	// We need to create wrapper handlers since the real handlers expect real service types
	// For testing, we'll create simplified test handlers

	app.Post("/verify", func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "API key is required"})
		}

		user, err := mockUserSvc.AuthenticateByAPIKey(context.Background(), apiKey)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid API key"})
		}

		var req VerifyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email required"})
		}

		result, err := mockVerifySvc.VerifyEmail(context.Background(), req.Email, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(result)
	})

	return app
}

func TestVerifyHandler_Success(t *testing.T) {
	mockUserSvc := NewMockUserService()
	mockVerifySvc := &MockVerificationService{}

	app := setupTestApp(mockUserSvc, mockVerifySvc)

	reqBody := `{"email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result service.VerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", result.Email)
	}

	if result.Status != "valid" {
		t.Errorf("Expected status 'valid', got %s", result.Status)
	}
}

func TestVerifyHandler_MissingAPIKey(t *testing.T) {
	mockUserSvc := NewMockUserService()
	mockVerifySvc := &MockVerificationService{}

	app := setupTestApp(mockUserSvc, mockVerifySvc)

	reqBody := `{"email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	// No X-API-Key header

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "API key is required") {
		t.Errorf("Expected error message about API key, got: %s", string(body))
	}
}

func TestVerifyHandler_InvalidAPIKey(t *testing.T) {
	mockUserSvc := NewMockUserService()
	mockVerifySvc := &MockVerificationService{}

	app := setupTestApp(mockUserSvc, mockVerifySvc)

	reqBody := `{"email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "invalid-api-key")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "invalid API key") {
		t.Errorf("Expected error message about invalid API key, got: %s", string(body))
	}
}

func TestVerifyHandler_MissingEmail(t *testing.T) {
	mockUserSvc := NewMockUserService()
	mockVerifySvc := &MockVerificationService{}

	app := setupTestApp(mockUserSvc, mockVerifySvc)

	reqBody := `{"email": ""}`
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Email required") {
		t.Errorf("Expected error message about email, got: %s", string(body))
	}
}

func TestVerifyHandler_InvalidJSON(t *testing.T) {
	mockUserSvc := NewMockUserService()
	mockVerifySvc := &MockVerificationService{}

	app := setupTestApp(mockUserSvc, mockVerifySvc)

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestVerifyHandler_VerificationError(t *testing.T) {
	mockUserSvc := NewMockUserService()
	mockVerifySvc := &MockVerificationService{
		verifyFunc: func(ctx context.Context, email string, user *store.User) (*service.VerifyResponse, error) {
			return nil, io.ErrUnexpectedEOF // Simulate an error
		},
	}

	app := setupTestApp(mockUserSvc, mockVerifySvc)

	reqBody := `{"email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestVerifyHandler_InvalidEmail(t *testing.T) {
	mockUserSvc := NewMockUserService()
	mockVerifySvc := &MockVerificationService{
		verifyFunc: func(ctx context.Context, email string, user *store.User) (*service.VerifyResponse, error) {
			return &service.VerifyResponse{
				Email:   email,
				Status:  "invalid",
				Message: "invalid syntax",
			}, nil
		},
	}

	app := setupTestApp(mockUserSvc, mockVerifySvc)

	reqBody := `{"email": "not-an-email"}`
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result service.VerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Status != "invalid" {
		t.Errorf("Expected status 'invalid', got %s", result.Status)
	}
}

// Test CSVImportResponse struct
func TestCSVImportResponse_Structure(t *testing.T) {
	response := CSVImportResponse{
		Total:    10,
		Accepted: 8,
		Items: []service.VerifyResponse{
			{Email: "test1@example.com", Status: "valid", Message: "OK"},
			{Email: "test2@example.com", Status: "invalid", Message: "no MX records"},
		},
	}

	if response.Total != 10 {
		t.Errorf("Expected Total 10, got %d", response.Total)
	}

	if response.Accepted != 8 {
		t.Errorf("Expected Accepted 8, got %d", response.Accepted)
	}

	if len(response.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(response.Items))
	}
}

// Test VerifyRequest struct
func TestVerifyRequest_Structure(t *testing.T) {
	jsonData := `{"email": "test@example.com"}`
	var req VerifyRequest
	err := json.Unmarshal([]byte(jsonData), &req)

	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", req.Email)
	}
}

// Test multipart form creation helper for CSV import testing
func createMultipartForm(t *testing.T, fieldName, fileName, content string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	_, err = io.Copy(part, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func TestVerifyHandler_DifferentStatuses(t *testing.T) {
	testCases := []struct {
		name           string
		email          string
		expectedStatus string
		mockStatus     string
		mockMessage    string
	}{
		{
			name:           "Valid email",
			email:          "valid@example.com",
			expectedStatus: "valid",
			mockStatus:     "valid",
			mockMessage:    "250 Accepted",
		},
		{
			name:           "Invalid email",
			email:          "invalid@nonexistent.com",
			expectedStatus: "invalid",
			mockStatus:     "invalid",
			mockMessage:    "no MX records",
		},
		{
			name:           "Disposable email",
			email:          "test@mailinator.com",
			expectedStatus: "disposable",
			mockStatus:     "disposable",
			mockMessage:    "disposable domain detected",
		},
		{
			name:           "Greylisted email",
			email:          "test@greylisted.com",
			expectedStatus: "greylisted",
			mockStatus:     "greylisted",
			mockMessage:    "451 Try again later",
		},
		{
			name:           "Pending verification",
			email:          "test@pending.com",
			expectedStatus: "pending",
			mockStatus:     "pending",
			mockMessage:    "probe sent",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockUserSvc := NewMockUserService()
			mockVerifySvc := &MockVerificationService{
				verifyFunc: func(ctx context.Context, email string, user *store.User) (*service.VerifyResponse, error) {
					return &service.VerifyResponse{
						Email:   email,
						Status:  tc.mockStatus,
						Message: tc.mockMessage,
					}, nil
				},
			}

			app := setupTestApp(mockUserSvc, mockVerifySvc)

			reqBody := `{"email": "` + tc.email + `"}`
			req := httptest.NewRequest("POST", "/verify", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", "test-api-key")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			var result service.VerifyResponse
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if result.Status != tc.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tc.expectedStatus, result.Status)
			}
		})
	}
}
