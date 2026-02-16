package handler

import (
	"github.com/gofiber/fiber/v2"
	"email-verifier-api/internal/verifier"
)

type VerifyRequest struct {
	Email string `json:"email"`
}

func VerifyHandler(v *verifier.EmailVerifier, apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Simple API Key Auth
		if c.Get("X-API-Key") != apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid API Key"})
		}

		var req VerifyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email required"})
		}

		result := v.Verify(req.Email)
		return c.JSON(result)
	}
}

// CheckTorHandler returns a handler that checks if traffic is routed through Tor.
func CheckTorHandler(v *verifier.EmailVerifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result := v.CheckTor()
		return c.JSON(result)
	}
}