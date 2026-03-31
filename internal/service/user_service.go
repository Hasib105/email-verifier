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
	"golang.org/x/crypto/bcrypt"
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
	Password   string `json:"password"`
	WebhookURL string `json:"webhook_url"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User   *store.User `json:"user"`
	APIKey string      `json:"api_key"`
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
	if req.Password == "" {
		return nil, errors.New("password is required")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user with this email already exists")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	// First user is superuser
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	isSuperuser := len(users) == 0

	input := store.UserInput{
		ID:           uuid.NewString(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		APIKey:       apiKey,
		WebhookURL:   req.WebhookURL,
		IsSuperuser:  isSuperuser,
		Active:       true,
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

func (s *UserService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}
	if !user.Active {
		return nil, errors.New("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	return &LoginResponse{
		User:   user,
		APIKey: user.APIKey,
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

func (s *UserService) UpdateUser(ctx context.Context, userID string, isSuperuser *bool) (*store.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	if isSuperuser != nil {
		if err := s.repo.UpdateUserSuperuser(ctx, userID, *isSuperuser); err != nil {
			return nil, err
		}
		user.IsSuperuser = *isSuperuser
	}

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	return s.repo.DeleteUser(ctx, userID)
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "evk_" + hex.EncodeToString(bytes), nil
}
