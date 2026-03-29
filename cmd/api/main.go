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

func main() {
	cfg := config.Load()

	// Initialize Verifier
	// Use a generic domain for EHLO to avoid rejection
	verifier := verifier.New(
		"verify@localhost",
		"localhost",
		cfg.TorSocksAddr,
		cfg.MaxConcurrency,
		cfg.Timeout,
	)

	verificationRepo, err := repo.New(cfg.ResolveDatabaseDSN())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer verificationRepo.Close()

	probeSender := service.NewSMTPProbeSender(verificationRepo, cfg.TorSocksAddr)

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
		},
	)

	go verificationService.StartScheduler(context.Background())

	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB for CSV imports
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// Routes
	app.Post("/verify", handler.VerifyHandler(verificationService, cfg.APIKey))
	app.Post("/verify/import-csv", handler.ImportCSVHandler(verificationService, cfg.APIKey))
	app.Post("/smtp-accounts", handler.CreateSMTPAccountHandler(verificationService, cfg.APIKey))
	app.Get("/smtp-accounts", handler.ListSMTPAccountsHandler(verificationService, cfg.APIKey))
	app.Post("/email-templates", handler.CreateEmailTemplateHandler(verificationService, cfg.APIKey))
	app.Get("/email-templates", handler.ListEmailTemplatesHandler(verificationService, cfg.APIKey))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/check-tor", handler.CheckTorHandler(verifier))

	log.Printf("Starting API on port %s with Tor at %s", cfg.Port, cfg.TorSocksAddr)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
