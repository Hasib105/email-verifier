package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	APIKey             string
	VerifierMailFrom   string
	VerifierEHLODomain string
	SMTPProxyPool      []string
	MaxConcurrency     int
	Timeout            time.Duration
	DatabaseDSN        string
	DBHost             string
	DBPort             int
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string

	WebhookURL     string
	WebhookTimeout time.Duration

	CheckInterval     time.Duration
	FirstBounceDelay  time.Duration
	SecondBounceDelay time.Duration
	HardResultTTL     time.Duration
	DirectValidTTL    time.Duration
	ProbeValidTTL     time.Duration
	TransientTTL      time.Duration
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "3000"),
		APIKey:             getEnv("API_KEY", "secret-key-change-me"),
		VerifierMailFrom:   getEnv("VERIFIER_MAIL_FROM", "verify@localhost"),
		VerifierEHLODomain: getEnv("VERIFIER_EHLO_DOMAIN", "localhost"),
		SMTPProxyPool:      getEnvList("SMTP_PROXY_POOL"),
		MaxConcurrency:     getEnvInt("MAX_CONCURRENCY", 10),
		Timeout:            getEnvDuration("TIMEOUT", 20*time.Second),
		DBHost:             getEnv("DB_HOST", "postgres"),
		DBPort:             getEnvInt("DB_PORT", 5432),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "postgres"),
		DBName:             getEnv("DB_NAME", "verifier"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),

		WebhookURL:     getEnv("WEBHOOK_URL", ""),
		WebhookTimeout: getEnvDuration("WEBHOOK_TIMEOUT", 10*time.Second),

		CheckInterval:     getEnvDuration("CHECK_INTERVAL", 15*time.Second),
		FirstBounceDelay:  getEnvDuration("FIRST_BOUNCE_DELAY", 1*time.Minute),
		SecondBounceDelay: getEnvDuration("SECOND_BOUNCE_DELAY", 6*time.Hour),
		HardResultTTL:     getEnvDuration("HARD_RESULT_TTL", 7*24*time.Hour),
		DirectValidTTL:    getEnvDuration("DIRECT_VALID_TTL", 72*time.Hour),
		ProbeValidTTL:     getEnvDuration("PROBE_VALID_TTL", 24*time.Hour),
		TransientTTL:      getEnvDuration("TRANSIENT_RESULT_TTL", 6*time.Hour),
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

func getEnvList(key string) []string {
	raw := strings.TrimSpace(getEnv(key, ""))
	if raw == "" {
		return nil
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
