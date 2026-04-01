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

// @Summary Get email template by ID
// @Description Returns a specific email template
// @Tags templates
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Template ID"
// @Success 200 {object} store.EmailTemplate
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /email-templates/{id} [get]
func GetEmailTemplateHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		template, err := svc.GetEmailTemplate(context.Background(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if template == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
		}

		return c.JSON(template)
	}
}

// @Summary Update email template
// @Description Updates an existing email template
// @Tags templates
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Template ID"
// @Param request body service.EmailTemplateCreateRequest true "Template details"
// @Success 200 {object} store.EmailTemplate
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /email-templates/{id} [put]
func UpdateEmailTemplateHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		var req service.EmailTemplateCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		tmpl, err := svc.UpdateEmailTemplate(context.Background(), id, req, user.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if tmpl == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
		}

		return c.JSON(tmpl)
	}
}

// @Summary Delete email template
// @Description Deletes an email template
// @Tags templates
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /email-templates/{id} [delete]
func DeleteEmailTemplateHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		if err := svc.DeleteEmailTemplate(context.Background(), id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "template deleted"})
	}
}

// @Summary Get SMTP account by ID
// @Description Returns a specific SMTP account
// @Tags smtp
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Account ID"
// @Success 200 {object} store.SMTPAccount
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /smtp-accounts/{id} [get]
func GetSMTPAccountHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		account, err := svc.GetSMTPAccount(context.Background(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if account == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "account not found"})
		}

		return c.JSON(account)
	}
}

// @Summary Update SMTP account
// @Description Updates an existing SMTP account
// @Tags smtp
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Account ID"
// @Param request body service.SMTPAccountCreateRequest true "Account details"
// @Success 200 {object} store.SMTPAccount
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /smtp-accounts/{id} [put]
func UpdateSMTPAccountHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		var req service.SMTPAccountCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		account, err := svc.UpdateSMTPAccount(context.Background(), id, req, user.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if account == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "account not found"})
		}

		return c.JSON(account)
	}
}

// @Summary Delete SMTP account
// @Description Deletes an SMTP account
// @Tags smtp
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Account ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /smtp-accounts/{id} [delete]
func DeleteSMTPAccountHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		if err := svc.DeleteSMTPAccount(context.Background(), id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "account deleted"})
	}
}

// @Summary List verifications
// @Description Lists email verifications for the authenticated user
// @Tags verification
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /verifications [get]
func ListVerificationsHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		items, err := svc.ListVerifications(context.Background(), user.ID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": items})
	}
}

// @Summary Get verification by ID
// @Description Returns a specific verification record
// @Tags verification
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Verification ID"
// @Success 200 {object} store.VerificationRecord
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /verifications/{id} [get]
func GetVerificationHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		record, err := svc.GetVerification(context.Background(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if record == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "verification not found"})
		}

		return c.JSON(record)
	}
}

// @Summary Delete verification
// @Description Deletes a specific verification record owned by the authenticated user
// @Tags verification
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param id path string true "Verification ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /verifications/{id} [delete]
func DeleteVerificationHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id := c.Params("id")
		err = svc.DeleteVerificationForUser(context.Background(), id, user.ID)
		if errors.Is(err, service.ErrVerificationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrVerificationForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "verification deleted"})
	}
}

// @Summary Get verification stats
// @Description Returns verification statistics for the authenticated user
// @Tags verification
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /verifications/stats [get]
func GetVerificationStatsHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		stats, err := svc.GetVerificationStats(context.Background(), user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		total := 0
		for _, v := range stats {
			total += v
		}

		return c.JSON(fiber.Map{
			"total":     total,
			"by_status": stats,
		})
	}
}

// Admin handlers for all models

// @Summary List all verifications (admin)
// @Description Returns all verifications for superusers
// @Tags admin
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/verifications [get]
func AdminListVerificationsHandler(svc *service.EmailVerificationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		items, err := svc.ListAllVerifications(context.Background(), limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": items})
	}
}

// @Summary List all SMTP accounts (admin)
// @Description Returns all SMTP accounts for superusers
// @Tags admin
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} map[string][]store.SMTPAccount
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/smtp-accounts [get]
func AdminListSMTPAccountsHandler(svc *service.EmailVerificationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accounts, err := svc.ListSMTPAccounts(context.Background(), "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"items": accounts})
	}
}

// @Summary List all email templates (admin)
// @Description Returns all email templates for superusers
// @Tags admin
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} map[string][]store.EmailTemplate
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/email-templates [get]
func AdminListEmailTemplatesHandler(svc *service.EmailVerificationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		templates, err := svc.ListEmailTemplates(context.Background(), "")
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
