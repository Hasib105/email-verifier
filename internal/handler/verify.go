package handler

import (
	"context"
	"email-verifier-api/internal/service"
	"email-verifier-api/internal/verifier"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io"
)

type VerifyRequest struct {
	Email string `json:"email"`
}

type CSVImportResponse struct {
	Total    int                      `json:"total"`
	Accepted int                      `json:"accepted"`
	Items    []service.VerifyResponse `json:"items"`
}

func checkAPIKey(c *fiber.Ctx, apiKey string) error {
	if c.Get("X-API-Key") != apiKey {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid API Key"})
	}
	return nil
}

func VerifyHandler(svc *service.EmailVerificationService, apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Simple API Key Auth
		if err := checkAPIKey(c, apiKey); err != nil {
			return err
		}

		var req VerifyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email required"})
		}

		result, err := svc.VerifyEmail(context.Background(), req.Email)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(result)
	}
}

func ImportCSVHandler(svc *service.EmailVerificationService, apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := checkAPIKey(c, apiKey); err != nil {
			return err
		}

		fileHeader, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "CSV file is required in form field 'file'"})
		}

		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot open uploaded file"})
		}
		defer file.Close()

		reader := csv.NewReader(file)
		reader.FieldsPerRecord = -1

		responses := make([]service.VerifyResponse, 0, 128)
		accepted := 0
		total := 0

		for {
			record, err := reader.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("invalid csv: %v", err)})
			}

			if len(record) == 0 {
				continue
			}

			total++
			email := record[0]
			res, err := svc.VerifyEmail(context.Background(), email)
			if err != nil {
				responses = append(responses, service.VerifyResponse{
					Email:   email,
					Status:  "error",
					Message: err.Error(),
				})
				continue
			}

			responses = append(responses, res)
			accepted++
		}

		return c.JSON(CSVImportResponse{
			Total:    total,
			Accepted: accepted,
			Items:    responses,
		})
	}
}

func CreateSMTPAccountHandler(svc *service.EmailVerificationService, apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := checkAPIKey(c, apiKey); err != nil {
			return err
		}

		var req service.SMTPAccountCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		account, err := svc.CreateSMTPAccount(context.Background(), req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(account)
	}
}

func ListSMTPAccountsHandler(svc *service.EmailVerificationService, apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := checkAPIKey(c, apiKey); err != nil {
			return err
		}

		accounts, err := svc.ListSMTPAccounts(context.Background())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": accounts})
	}
}

func CreateEmailTemplateHandler(svc *service.EmailVerificationService, apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := checkAPIKey(c, apiKey); err != nil {
			return err
		}

		var req service.EmailTemplateCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		tmpl, err := svc.CreateEmailTemplate(context.Background(), req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(tmpl)
	}
}

func ListEmailTemplatesHandler(svc *service.EmailVerificationService, apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := checkAPIKey(c, apiKey); err != nil {
			return err
		}

		templates, err := svc.ListEmailTemplates(context.Background())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": templates})
	}
}

// CheckTorHandler returns a handler that checks if traffic is routed through Tor.
func CheckTorHandler(v *verifier.EmailVerifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result := v.CheckTor()
		return c.JSON(result)
	}
}
