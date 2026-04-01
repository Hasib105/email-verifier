package handler

import (
	"context"
	"email-verifier-api/internal/service"
	"email-verifier-api/internal/store"
	"encoding/csv"
	"errors"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
)

const maxBatchVerifyEmails = 1000

type VerifyRequest struct {
	Email string `json:"email"`
}

type BatchVerifyRequest struct {
	Emails []string `json:"emails"`
}

type BatchVerifyResponse struct {
	Total    int                      `json:"total"`
	Accepted int                      `json:"accepted"`
	Items    []service.VerifyResponse `json:"items"`
}

type CSVImportResponse struct {
	Total    int                      `json:"total"`
	Accepted int                      `json:"accepted"`
	Items    []service.VerifyResponse `json:"items"`
}

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

func VerifyHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		var req VerifyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
		}
		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
		}

		result, err := svc.VerifyEmail(context.Background(), req.Email, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	}
}

func BatchVerifyHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		var req BatchVerifyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
		}
		if len(req.Emails) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "emails must contain at least one item"})
		}
		if len(req.Emails) > maxBatchVerifyEmails {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("batch limit exceeded: max %d emails", maxBatchVerifyEmails)})
		}

		items, accepted := svc.VerifyEmailBatch(context.Background(), req.Emails, user)
		return c.JSON(BatchVerifyResponse{Total: len(req.Emails), Accepted: accepted, Items: items})
	}
}

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

		items := make([]service.VerifyResponse, 0, 128)
		total := 0
		accepted := 0

		for {
			record, err := reader.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("invalid csv: %v", err)})
			}
			if len(record) == 0 {
				continue
			}

			total++
			result, err := svc.VerifyEmail(context.Background(), record[0], user)
			if err != nil {
				items = append(items, service.VerifyResponse{
					Email:             record[0],
					Classification:    "unknown",
					ConfidenceScore:   0,
					RiskLevel:         "high",
					Deterministic:     false,
					State:             "completed",
					ReasonCodes:       []string{"verification_error"},
					ProtocolSummary:   err.Error(),
					EnrichmentSummary: "",
				})
				continue
			}
			items = append(items, result)
			accepted++
		}

		return c.JSON(CSVImportResponse{
			Total:    total,
			Accepted: accepted,
			Items:    items,
		})
	}
}

func ListVerificationsHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		items, err := svc.ListVerifications(context.Background(), user.ID, c.QueryInt("limit", 50), c.QueryInt("offset", 0))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"items": items})
	}
}

func GetVerificationHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		base, err := svc.GetVerification(context.Background(), c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if base == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "verification not found"})
		}
		if !user.IsSuperuser && base.UserID != user.ID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "verification access denied"})
		}

		record, err := svc.GetVerificationDetail(context.Background(), c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(record)
	}
}

func DeleteVerificationHandler(svc *service.EmailVerificationService, userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, userSvc)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		err = svc.DeleteVerificationForUser(context.Background(), c.Params("id"), user.ID)
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
		for _, count := range stats {
			total += count
		}

		return c.JSON(fiber.Map{
			"total":             total,
			"by_classification": stats,
		})
	}
}

func AdminListVerificationsHandler(svc *service.EmailVerificationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := svc.ListAllVerifications(context.Background(), c.QueryInt("limit", 50), c.QueryInt("offset", 0))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"items": items})
	}
}
