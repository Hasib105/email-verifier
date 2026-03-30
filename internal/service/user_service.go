package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"email-verifier-api/internal/repo"
	"email-verifier-api/internal/store"

	"github.com/google/uuid"
)

type UserService struct {
	repo *repo.Repository
}

func NewUserService(r *repo.Repository) *UserService {
	return &UserService{repo: r}
}

type SignupRequest struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	WebhookURL string `json:"webhook_url"`
}

type SignupResponse struct {
	User   *store.User `json:"user"`
	APIKey string      `json:"api_key"`
}

func (s *UserService) Signup(ctx context.Context, req SignupRequest) (*SignupResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.WebhookURL = strings.TrimSpace(req.WebhookURL)

	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if !strings.Contains(req.Email, "@") {
		return nil, errors.New("invalid email format")
	}

	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user with this email already exists")
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	input := store.UserInput{
		ID:         uuid.NewString(),
		Name:       req.Name,
		Email:      req.Email,
		APIKey:     apiKey,
		WebhookURL: req.WebhookURL,
		Active:     true,
	}

	user, err := s.repo.CreateUser(ctx, input)
	if err != nil {
		return nil, err
	}

	return &SignupResponse{
		User:   user,
		APIKey: apiKey,
	}, nil
}

func (s *UserService) AuthenticateByAPIKey(ctx context.Context, apiKey string) (*store.User, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}
	return s.repo.GetUserByAPIKey(ctx, apiKey)
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *UserService) UpdateWebhook(ctx context.Context, userID, webhookURL string) error {
	return s.repo.UpdateUserWebhook(ctx, userID, webhookURL)
}

func (s *UserService) ListUsers(ctx context.Context) ([]store.User, error) {
	return s.repo.ListUsers(ctx)
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "evk_" + hex.EncodeToString(bytes), nil
}
