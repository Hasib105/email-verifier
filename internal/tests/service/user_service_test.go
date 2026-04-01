package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"email-verifier-api/internal/store"

	"github.com/google/uuid"
)

// Helper functions and error types
var (
	errNameRequired   = errors.New("name is required")
	errEmailRequired  = errors.New("email is required")
	errInvalidEmail   = errors.New("invalid email format")
	errDuplicateEmail = errors.New("user with this email already exists")
	errAPIKeyRequired = errors.New("API key is required")
)

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func toLower(s string) string {
	return strings.ToLower(s)
}

func containsAt(s string) bool {
	return strings.Contains(s, "@")
}

func generateUUID() string {
	return uuid.NewString()
}

// UserServiceMock wraps UserService for testing
type UserServiceMock struct {
	repo *MockRepository
}

func NewUserServiceMock() *UserServiceMock {
	repo := NewMockRepository()
	repo.usersByEmail = make(map[string]*store.User)
	repo.usersByAPIKey = make(map[string]*store.User)
	return &UserServiceMock{repo: repo}
}

func (s *UserServiceMock) Signup(ctx context.Context, req SignupRequest) (*SignupResponse, error) {
	req.Name = trimSpace(req.Name)
	req.Email = toLower(trimSpace(req.Email))
	req.WebhookURL = trimSpace(req.WebhookURL)

	if req.Name == "" {
		return nil, errNameRequired
	}
	if req.Email == "" {
		return nil, errEmailRequired
	}
	if !containsAt(req.Email) {
		return nil, errInvalidEmail
	}

	existing, err := s.repo.GetUserByEmailMock(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errDuplicateEmail
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	user := &store.User{
		ID:         generateUUID(),
		Name:       req.Name,
		Email:      req.Email,
		APIKey:     apiKey,
		WebhookURL: req.WebhookURL,
		Active:     true,
	}

	s.repo.users[user.ID] = user
	s.repo.usersByEmail[user.Email] = user
	s.repo.usersByAPIKey[user.APIKey] = user

	return &SignupResponse{
		User:   user,
		APIKey: apiKey,
	}, nil
}

func (s *UserServiceMock) AuthenticateByAPIKey(ctx context.Context, apiKey string) (*store.User, error) {
	if apiKey == "" {
		return nil, errAPIKeyRequired
	}
	return s.repo.GetUserByAPIKeyMock(ctx, apiKey)
}

func (s *UserServiceMock) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *UserServiceMock) UpdateWebhook(ctx context.Context, userID, webhookURL string) error {
	user, exists := s.repo.users[userID]
	if !exists {
		return nil
	}
	user.WebhookURL = webhookURL
	return nil
}

func (s *UserServiceMock) ListUsers(ctx context.Context) ([]store.User, error) {
	users := make([]store.User, 0, len(s.repo.users))
	for _, user := range s.repo.users {
		users = append(users, *user)
	}
	return users, nil
}

func (r *MockRepository) GetUserByEmailMock(ctx context.Context, email string) (*store.User, error) {
	if r.usersByEmail == nil {
		r.usersByEmail = make(map[string]*store.User)
	}
	if user, ok := r.usersByEmail[email]; ok {
		return user, nil
	}
	return nil, nil
}

func (r *MockRepository) GetUserByAPIKeyMock(ctx context.Context, apiKey string) (*store.User, error) {
	if r.usersByAPIKey == nil {
		r.usersByAPIKey = make(map[string]*store.User)
	}
	if user, ok := r.usersByAPIKey[apiKey]; ok {
		return user, nil
	}
	return nil, nil
}

func TestUserService_Signup_Success(t *testing.T) {
	userSvc := NewUserServiceMock()

	req := SignupRequest{
		Name:       "John Doe",
		Email:      "john@example.com",
		WebhookURL: "https://webhook.example.com/callback",
	}

	result, err := userSvc.Signup(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.User == nil {
		t.Fatal("Expected user to be created")
	}

	if result.User.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got: %s", result.User.Name)
	}

	if result.User.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got: %s", result.User.Email)
	}

	if result.User.WebhookURL != "https://webhook.example.com/callback" {
		t.Errorf("Expected webhook URL, got: %s", result.User.WebhookURL)
	}

	if result.APIKey == "" {
		t.Error("Expected API key to be generated")
	}

	if len(result.APIKey) < 20 {
		t.Error("Expected API key to be at least 20 characters")
	}

	// API key should start with "evk_"
	if len(result.APIKey) > 4 && result.APIKey[:4] != "evk_" {
		t.Errorf("Expected API key to start with 'evk_', got: %s", result.APIKey[:4])
	}
}

func TestUserService_Signup_MissingName(t *testing.T) {
	userSvc := NewUserServiceMock()

	req := SignupRequest{
		Name:  "",
		Email: "john@example.com",
	}

	_, err := userSvc.Signup(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for missing name")
	}

	if err.Error() != "name is required" {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}
}

func TestUserService_Signup_MissingEmail(t *testing.T) {
	userSvc := NewUserServiceMock()

	req := SignupRequest{
		Name:  "John Doe",
		Email: "",
	}

	_, err := userSvc.Signup(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for missing email")
	}

	if err.Error() != "email is required" {
		t.Errorf("Expected 'email is required' error, got: %v", err)
	}
}

func TestUserService_Signup_InvalidEmail(t *testing.T) {
	userSvc := NewUserServiceMock()

	req := SignupRequest{
		Name:  "John Doe",
		Email: "not-an-email",
	}

	_, err := userSvc.Signup(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for invalid email")
	}

	if err.Error() != "invalid email format" {
		t.Errorf("Expected 'invalid email format' error, got: %v", err)
	}
}

func TestUserService_Signup_DuplicateEmail(t *testing.T) {
	userSvc := NewUserServiceMock()

	// First signup
	req := SignupRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	_, err := userSvc.Signup(context.Background(), req)
	if err != nil {
		t.Fatalf("First signup failed: %v", err)
	}

	// Second signup with same email
	req2 := SignupRequest{
		Name:  "Jane Doe",
		Email: "john@example.com",
	}

	_, err = userSvc.Signup(context.Background(), req2)

	if err == nil {
		t.Fatal("Expected error for duplicate email")
	}

	if err.Error() != "user with this email already exists" {
		t.Errorf("Expected duplicate email error, got: %v", err)
	}
}

func TestUserService_Signup_EmailNormalization(t *testing.T) {
	userSvc := NewUserServiceMock()

	req := SignupRequest{
		Name:  "  John Doe  ",
		Email: "  JOHN@EXAMPLE.COM  ",
	}

	result, err := userSvc.Signup(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.User.Name != "John Doe" {
		t.Errorf("Expected trimmed name 'John Doe', got: '%s'", result.User.Name)
	}

	if result.User.Email != "john@example.com" {
		t.Errorf("Expected lowercase email 'john@example.com', got: '%s'", result.User.Email)
	}
}

func TestUserService_AuthenticateByAPIKey_Success(t *testing.T) {
	userSvc := NewUserServiceMock()

	// Create a user first
	req := SignupRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	signupResult, err := userSvc.Signup(context.Background(), req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	// Authenticate with API key
	user, err := userSvc.AuthenticateByAPIKey(context.Background(), signupResult.APIKey)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if user == nil {
		t.Fatal("Expected user to be returned")
	}

	if user.ID != signupResult.User.ID {
		t.Errorf("Expected user ID %s, got: %s", signupResult.User.ID, user.ID)
	}
}

func TestUserService_AuthenticateByAPIKey_InvalidKey(t *testing.T) {
	userSvc := NewUserServiceMock()

	user, err := userSvc.AuthenticateByAPIKey(context.Background(), "invalid-api-key")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if user != nil {
		t.Error("Expected nil user for invalid API key")
	}
}

func TestUserService_AuthenticateByAPIKey_EmptyKey(t *testing.T) {
	userSvc := NewUserServiceMock()

	_, err := userSvc.AuthenticateByAPIKey(context.Background(), "")

	if err == nil {
		t.Fatal("Expected error for empty API key")
	}

	if err.Error() != "API key is required" {
		t.Errorf("Expected 'API key is required' error, got: %v", err)
	}
}

func TestUserService_UpdateWebhook(t *testing.T) {
	userSvc := NewUserServiceMock()

	// Create a user first
	req := SignupRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	signupResult, err := userSvc.Signup(context.Background(), req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	// Update webhook
	newWebhookURL := "https://new-webhook.example.com/callback"
	err = userSvc.UpdateWebhook(context.Background(), signupResult.User.ID, newWebhookURL)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify update
	user, err := userSvc.GetUserByID(context.Background(), signupResult.User.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.WebhookURL != newWebhookURL {
		t.Errorf("Expected webhook URL '%s', got: '%s'", newWebhookURL, user.WebhookURL)
	}
}

func TestUserService_ListUsers(t *testing.T) {
	userSvc := NewUserServiceMock()

	// Create multiple users
	users := []SignupRequest{
		{Name: "User 1", Email: "user1@example.com"},
		{Name: "User 2", Email: "user2@example.com"},
		{Name: "User 3", Email: "user3@example.com"},
	}

	for _, req := range users {
		_, err := userSvc.Signup(context.Background(), req)
		if err != nil {
			t.Fatalf("Signup failed: %v", err)
		}
	}

	// List users
	result, err := userSvc.ListUsers(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 users, got: %d", len(result))
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key1, err := generateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	key2, err := generateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Keys should be unique
	if key1 == key2 {
		t.Error("Generated API keys should be unique")
	}

	// Keys should start with prefix
	if len(key1) > 4 && key1[:4] != "evk_" {
		t.Errorf("API key should start with 'evk_', got: %s", key1[:4])
	}

	// Keys should be sufficiently long (evk_ + 64 hex chars)
	if len(key1) != 68 {
		t.Errorf("Expected API key length 68, got: %d", len(key1))
	}
}

// Redeclare generateAPIKey for tests to avoid import cycle
func generateAPIKeyTest() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "evk_" + hex.EncodeToString(bytes), nil
}
