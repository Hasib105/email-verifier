package handler

import (
	"context"
	"email-verifier-api/internal/service"
	"email-verifier-api/internal/store"
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

// authenticateUser extracts API key from header and returns the user
func authenticateUser(c *fiber.Ctx, userSvc *service.UserService) (*store.User, error) {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	user, err := userSvc.AuthenticateByAPIKey(context.Background(), apiKey)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid API key")
	}

	return user, nil
}

// @Summary Verify an email address
// @Description Verifies if an email address is deliverable using SMTP checks
// @Tags verification
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param request body VerifyRequest true "Email to verify"
// @Success 200 {object} service.VerifyResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /verify [post]
func VerifyHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		var req VerifyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email required"})
		}

		result, err := svc.VerifyEmail(context.Background(), req.Email, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(result)
	}
}

// @Summary Import emails from CSV
// @Description Imports and verifies multiple emails from a CSV file
// @Tags verification
// @Accept multipart/form-data
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param file formData file true "CSV file with emails"
// @Success 200 {object} CSVImportResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /verify/import-csv [post]
func ImportCSVHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
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
			res, err := svc.VerifyEmail(context.Background(), email, user)
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

// @Summary Create SMTP account
// @Description Creates a new SMTP account for sending probe emails
// @Tags smtp
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param request body service.SMTPAccountCreateRequest true "SMTP account details"
// @Success 201 {object} store.SMTPAccount
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /smtp-accounts [post]
func CreateSMTPAccountHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		var req service.SMTPAccountCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		account, err := svc.CreateSMTPAccount(context.Background(), req, user.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(account)
	}
}

// @Summary List SMTP accounts
// @Description Lists all SMTP accounts for the authenticated user
// @Tags smtp
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} map[string][]store.SMTPAccount
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /smtp-accounts [get]
func ListSMTPAccountsHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		accounts, err := svc.ListSMTPAccounts(context.Background(), user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": accounts})
	}
}

// @Summary Create email template
// @Description Creates a new email template for probe emails
// @Tags templates
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param request body service.EmailTemplateCreateRequest true "Template details"
// @Success 201 {object} store.EmailTemplate
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /email-templates [post]
func CreateEmailTemplateHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		var req service.EmailTemplateCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		tmpl, err := svc.CreateEmailTemplate(context.Background(), req, user.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(tmpl)
	}
}

// @Summary List email templates
// @Description Lists all email templates for the authenticated user
// @Tags templates
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} map[string][]store.EmailTemplate
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /email-templates [get]
func ListEmailTemplatesHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		templates, err := svc.ListEmailTemplates(context.Background(), user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": templates})
	}
}

// @Summary Check Tor connectivity
// @Description Checks if the API is properly routing traffic through Tor
// @Tags health
// @Produce json
// @Success 200 {object} verifier.TorCheckResult
// @Router /check-tor [get]
func CheckTorHandler(v *verifier.EmailVerifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result := v.CheckTor()
		return c.JSON(result)
	}
}

// @Summary Update webhook URL
// @Description Updates the webhook URL for the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param request body UpdateWebhookRequest true "Webhook URL"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/webhook [put]
func UpdateWebhookHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		var req UpdateWebhookRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if err := userSvc.UpdateWebhook(context.Background(), user.ID, req.WebhookURL); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Webhook URL updated successfully"})
	}
}

type UpdateWebhookRequest struct {
	WebhookURL string `json:"webhook_url"`
}

// @Summary Get current user
// @Description Returns the authenticated user's information
// @Tags users
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} store.User
// @Failure 401 {object} map[string]string
// @Router /users/me [get]
func GetCurrentUserHandler(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(user)
	}
}
