package main

import (
	"log"
	"email-verifier-api/internal/config"
	"email-verifier-api/internal/handler"
	"email-verifier-api/internal/verifier"

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

	app := fiber.New(fiber.Config{
		BodyLimit: 4 * 1024, // 4KB limit
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// Routes
	app.Post("/verify", handler.VerifyHandler(verifier, cfg.APIKey))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/check-tor", handler.CheckTorHandler(verifier))

	log.Printf("Starting API on port %s with Tor at %s", cfg.Port, cfg.TorSocksAddr)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}