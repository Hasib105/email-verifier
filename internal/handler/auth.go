package handler

import (
	"context"

	"email-verifier-api/internal/service"

	"github.com/gofiber/fiber/v2"
)

// @Summary Regenerate User API Key
// @Description Regenerates the API key for the current user
// @Tags users
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/api-key/regenerate [post]
func RegenerateAPIKeyHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		newKey, err := userSvc.RegenerateAPIKey(context.Background(), user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"api_key": newKey})
	}
}

// @Summary Register a new user
// @Description Creates a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body service.SignupRequest true "User registration details"
// @Success 201 {object} service.SignupResponse
// @Failure 400 {object} map[string]string
// @Router /auth/register [post]
func RegisterHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req service.SignupRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		resp, err := userSvc.Signup(context.Background(), req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

// @Summary Login user
// @Description Authenticates a user and returns API key
// @Tags auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Login credentials"
// @Success 200 {object} service.LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func LoginHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req service.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		resp, err := userSvc.Login(context.Background(), req)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(resp)
	}
}

// requireSuperuser is middleware that checks if user is a superuser
func RequireSuperuser(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		if !user.IsSuperuser {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "superuser access required"})
		}

		c.Locals("user", user)
		return c.Next()
	}
}

// @Summary List all users (admin only)
// @Description Returns list of all users for superusers only
// @Tags admin
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} map[string][]store.User
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/users [get]
func AdminListUsersHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		users, err := userSvc.ListUsers(context.Background())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": users})
	}
}

type AdminUpdateUserRequest struct {
	IsSuperuser *bool `json:"is_superuser,omitempty"`
}

// @Summary Update user (admin only)
// @Description Updates user details like superuser status
// @Tags admin
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "User ID"
// @Param request body AdminUpdateUserRequest true "User update details"
// @Success 200 {object} store.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/users/{id} [put]
func AdminUpdateUserHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		var req AdminUpdateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		user, err := userSvc.UpdateUser(context.Background(), id, req.IsSuperuser)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(user)
	}
}

// @Summary Delete user (admin only)
// @Description Deletes a user account
// @Tags admin
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/users/{id} [delete]
func AdminDeleteUserHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		if err := userSvc.DeleteUser(context.Background(), id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "User deleted successfully"})
	}
}

// @Summary Delete verification (admin only)
// @Description Deletes a verification record
// @Tags admin
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Verification ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /admin/verifications/{id} [delete]
func AdminDeleteVerificationHandler(verificationSvc *service.EmailVerificationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		if err := verificationSvc.AdminDeleteVerification(context.Background(), id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Verification deleted successfully"})
	}
}

// @Summary Test webhook
// @Description Sends a test webhook to the specified URL
// @Tags settings
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param request body map[string]string true "Webhook URL"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /users/webhook/test [post]
func TestWebhookHandler(verificationSvc *service.EmailVerificationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if req.WebhookURL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "webhook_url is required"})
		}

		if err := verificationSvc.SendTestWebhook(context.Background(), req.WebhookURL); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Test webhook sent successfully"})
	}
}
