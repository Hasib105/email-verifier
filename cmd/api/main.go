package main

import (
	"context"
	"email-verifier-api/internal/config"
	"email-verifier-api/internal/handler"
	"email-verifier-api/internal/repo"
	"email-verifier-api/internal/service"
	"email-verifier-api/internal/verifier"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// @title Email Verifier API
// @version 1.0
// @description API for verifying email addresses using SMTP checks and bounce detection
// @host localhost:3000
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	cfg := config.Load()

	// Initialize Verifier
	verifier := verifier.New(
		cfg.VerifierMailFrom,
		cfg.VerifierEHLODomain,
		cfg.MaxConcurrency,
		cfg.Timeout,
	)

	verificationRepo, err := repo.New(cfg.ResolveDatabaseDSN())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer verificationRepo.Close()

	userService := service.NewUserService(verificationRepo)

	probeSender := service.NewSMTPProbeSender(verificationRepo)

	bounceChecker := service.NewIMAPBounceChecker()

	webhook := service.NewHTTPWebhookDispatcher(cfg.WebhookURL, cfg.WebhookTimeout)

	verificationService := service.NewEmailVerificationService(
		verifier,
		verificationRepo,
		probeSender,
		bounceChecker,
		webhook,
		service.ServiceConfig{
			FirstBounceDelay:  cfg.FirstBounceDelay,
			SecondBounceDelay: cfg.SecondBounceDelay,
			CheckInterval:     cfg.CheckInterval,
			HardResultTTL:     cfg.HardResultTTL,
			DirectValidTTL:    cfg.DirectValidTTL,
			ProbeValidTTL:     cfg.ProbeValidTTL,
			TransientTTL:      cfg.TransientTTL,
		},
	)

	go verificationService.StartScheduler(context.Background())

	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB for CSV imports
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// Health endpoints
	app.Get("/health", func(c *fiber.Ctx) error {
		health := verifier.HealthSnapshot()
		return c.JSON(fiber.Map{
			"status":               "ok",
			"mode":                 "v1-hardened",
			"direct_smtp_status":   health.DirectSMTPStatus,
			"last_checked_at":      health.LastCheckedAt,
			"message":              health.Message,
			"verifier_mail_from":   cfg.VerifierMailFrom,
			"verifier_ehlo_domain": cfg.VerifierEHLODomain,
		})
	})

	// Auth routes (no auth required)
	app.Post("/auth/register", handler.RegisterHandler(userService))
	app.Post("/auth/login", handler.LoginHandler(userService))

	// User management
	app.Get("/users/me", handler.GetCurrentUserHandler(userService))
	app.Put("/users/webhook", handler.UpdateWebhookHandler(userService))
	app.Post("/users/webhook/test", handler.TestWebhookHandler(verificationService))

	// Verification routes
	app.Post("/verify", handler.VerifyHandler(verificationService, userService))
	app.Post("/verify/batch", handler.BatchVerifyHandler(verificationService, userService))
	app.Post("/verify/import-csv", handler.ImportCSVHandler(verificationService, userService))
	app.Get("/verifications", handler.ListVerificationsHandler(verificationService, userService))
	app.Get("/verifications/stats", handler.GetVerificationStatsHandler(verificationService, userService))
	app.Get("/verifications/:id", handler.GetVerificationHandler(verificationService, userService))
	app.Delete("/verifications/:id", handler.DeleteVerificationHandler(verificationService, userService))

	// SMTP account routes
	app.Post("/smtp-accounts", handler.CreateSMTPAccountHandler(verificationService, userService))
	app.Get("/smtp-accounts", handler.ListSMTPAccountsHandler(verificationService, userService))
	app.Get("/smtp-accounts/:id", handler.GetSMTPAccountHandler(verificationService, userService))
	app.Put("/smtp-accounts/:id", handler.UpdateSMTPAccountHandler(verificationService, userService))
	app.Delete("/smtp-accounts/:id", handler.DeleteSMTPAccountHandler(verificationService, userService))

	// Email template routes
	app.Post("/email-templates", handler.CreateEmailTemplateHandler(verificationService, userService))
	app.Get("/email-templates", handler.ListEmailTemplatesHandler(verificationService, userService))
	app.Get("/email-templates/:id", handler.GetEmailTemplateHandler(verificationService, userService))
	app.Put("/email-templates/:id", handler.UpdateEmailTemplateHandler(verificationService, userService))
	app.Delete("/email-templates/:id", handler.DeleteEmailTemplateHandler(verificationService, userService))

	// Admin routes (superuser only)
	admin := app.Group("/admin", handler.RequireSuperuser(userService))
	admin.Get("/users", handler.AdminListUsersHandler(userService))
	admin.Put("/users/:id", handler.AdminUpdateUserHandler(userService))
	admin.Delete("/users/:id", handler.AdminDeleteUserHandler(userService))
	admin.Get("/verifications", handler.AdminListVerificationsHandler(verificationService))
	admin.Delete("/verifications/:id", handler.AdminDeleteVerificationHandler(verificationService))
	admin.Get("/smtp-accounts", handler.AdminListSMTPAccountsHandler(verificationService))
	admin.Get("/email-templates", handler.AdminListEmailTemplatesHandler(verificationService))

	log.Printf("Starting API on port %s with direct SMTP + bounce fallback", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
