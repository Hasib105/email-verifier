package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	APIKey             string
	MaxConcurrency     int
	Timeout            time.Duration
	DatabaseDSN        string
	DBHost             string
	DBPort             int
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	VerifierMailFrom   string
	VerifierEHLODomain string
	DeliverableTTL     time.Duration
	UndeliverableTTL   time.Duration
	AcceptAllTTL       time.Duration
	UnknownTTL         time.Duration
	DomainBaselineTTL  time.Duration
	EnrichmentWorkers  int
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "3000"),
		APIKey:             getEnv("API_KEY", "secret-key-change-me"),
		MaxConcurrency:     getEnvInt("MAX_CONCURRENCY", 10),
		Timeout:            getEnvDuration("TIMEOUT", 20*time.Second),
		DBHost:             getEnv("DB_HOST", "postgres"),
		DBPort:             getEnvInt("DB_PORT", 5432),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "postgres"),
		DBName:             getEnv("DB_NAME", "verifier"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		VerifierMailFrom:   getEnv("VERIFIER_MAIL_FROM", "verify@verifier.local"),
		VerifierEHLODomain: getEnv("VERIFIER_EHLO_DOMAIN", "verifier.local"),
		DeliverableTTL:     getEnvDuration("DELIVERABLE_TTL", 7*24*time.Hour),
		UndeliverableTTL:   getEnvDuration("UNDELIVERABLE_TTL", 24*time.Hour),
		AcceptAllTTL:       getEnvDuration("ACCEPT_ALL_TTL", 24*time.Hour),
		UnknownTTL:         getEnvDuration("UNKNOWN_TTL", 6*time.Hour),
		DomainBaselineTTL:  getEnvDuration("DOMAIN_BASELINE_TTL", 24*time.Hour),
		EnrichmentWorkers:  getEnvInt("ENRICHMENT_WORKERS", 2),
	}
}

func (c *Config) ResolveDatabaseDSN() string {
	if dsn := getEnv("DATABASE_DSN", ""); dsn != "" {
		return dsn
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return defaultVal
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return value
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	raw := getEnv(key, "")
	if raw == "" {
		return defaultVal
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return defaultVal
	}
	return value
}
