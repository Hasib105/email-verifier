package handler

import (
	"context"

	"email-verifier-api/internal/service"

	"github.com/gofiber/fiber/v2"
)

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

func RegisterHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req service.SignupRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
		}

		resp, err := userSvc.Signup(context.Background(), req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

func LoginHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req service.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
		}

		resp, err := userSvc.Login(context.Background(), req)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(resp)
	}
}

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

func AdminUpdateUserHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req AdminUpdateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
		}

		user, err := userSvc.UpdateUser(context.Background(), c.Params("id"), req.IsSuperuser)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(user)
	}
}

func AdminDeleteUserHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := userSvc.DeleteUser(context.Background(), c.Params("id")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "user deleted successfully"})
	}
}

func AdminDeleteVerificationHandler(verificationSvc *service.EmailVerificationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := verificationSvc.AdminDeleteVerification(context.Background(), c.Params("id")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "verification deleted successfully"})
	}
}

func GetCurrentUserHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(user)
	}
}
