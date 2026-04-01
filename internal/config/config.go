package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	APIKey         string
	Host           string
	TorSocksAddr   string
	TorControlAddr string
	MaxConcurrency int
	Timeout        time.Duration
	DatabaseDSN    string
	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string

	WebhookURL     string
	WebhookTimeout time.Duration

	CheckInterval     time.Duration
	FirstBounceDelay  time.Duration
	SecondBounceDelay time.Duration
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "3000"),
		APIKey:         getEnv("API_KEY", "secret-key-change-me"),
		TorSocksAddr:   getEnv("TOR_SOCKS_ADDR", "tor:9050"), // 'tor' is docker service name
		TorControlAddr: getEnv("TOR_CONTROL_ADDR", ""),       // e.g., "tor:9051"
		MaxConcurrency: getEnvInt("MAX_CONCURRENCY", 5),      // Tor is slow, keep this low
		Timeout:        getEnvDuration("TIMEOUT", 45*time.Second),
		DBHost:         getEnv("DB_HOST", "postgres"),
		DBPort:         getEnvInt("DB_PORT", 5432),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "verifier"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),

		WebhookURL:     getEnv("WEBHOOK_URL", ""),
		WebhookTimeout: getEnvDuration("WEBHOOK_TIMEOUT", 10*time.Second),

		CheckInterval:     getEnvDuration("CHECK_INTERVAL", 1*time.Minute),
		FirstBounceDelay:  getEnvDuration("FIRST_BOUNCE_DELAY", 1*time.Minute),
		SecondBounceDelay: getEnvDuration("SECOND_BOUNCE_DELAY", 6*time.Hour),
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
