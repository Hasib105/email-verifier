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

	verifierEngine := verifier.New(
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
	enrichmentService := service.NewEnrichmentService()

	verificationService := service.NewEmailVerificationService(
		verifierEngine,
		verificationRepo,
		enrichmentService,
		service.ServiceConfig{
			DeliverableTTL:    cfg.DeliverableTTL,
			UndeliverableTTL:  cfg.UndeliverableTTL,
			AcceptAllTTL:      cfg.AcceptAllTTL,
			UnknownTTL:        cfg.UnknownTTL,
			DomainBaselineTTL: cfg.DomainBaselineTTL,
			EnrichmentWorkers: cfg.EnrichmentWorkers,
		},
	)

	verificationService.StartBackground(context.Background())

	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB for CSV imports
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// Health endpoints
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":          "ok",
			"mode":            "direct-smtp-callout",
			"mail_from":       cfg.VerifierMailFrom,
			"ehlo_domain":     cfg.VerifierEHLODomain,
			"max_parallel":    cfg.MaxConcurrency,
			"baseline_ttl":    cfg.DomainBaselineTTL.String(),
			"deliverable_ttl": cfg.DeliverableTTL.String(),
		})
	})

	// Auth routes (no auth required)
	app.Post("/auth/register", handler.RegisterHandler(userService))
	app.Post("/auth/login", handler.LoginHandler(userService))

	// User management
	app.Get("/users/me", handler.GetCurrentUserHandler(userService))
	app.Post("/users/api-key/regenerate", handler.RegenerateAPIKeyHandler(userService))

	// Verification routes
	app.Post("/verifications", handler.VerifyHandler(verificationService, userService))
	app.Post("/verifications/batch", handler.BatchVerifyHandler(verificationService, userService))
	app.Post("/verifications/import-csv", handler.ImportCSVHandler(verificationService, userService))
	app.Get("/verifications", handler.ListVerificationsHandler(verificationService, userService))
	app.Get("/verifications/stats", handler.GetVerificationStatsHandler(verificationService, userService))
	app.Get("/verifications/:id", handler.GetVerificationHandler(verificationService, userService))
	app.Delete("/verifications/:id", handler.DeleteVerificationHandler(verificationService, userService))

	// Admin routes (superuser only)
	admin := app.Group("/admin", handler.RequireSuperuser(userService))
	admin.Get("/users", handler.AdminListUsersHandler(userService))
	admin.Put("/users/:id", handler.AdminUpdateUserHandler(userService))
	admin.Delete("/users/:id", handler.AdminDeleteUserHandler(userService))
	admin.Get("/verifications", handler.AdminListVerificationsHandler(verificationService))
	admin.Delete("/verifications/:id", handler.AdminDeleteVerificationHandler(verificationService))

	log.Printf("Starting API on port %s in V2 direct-callout mode", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
