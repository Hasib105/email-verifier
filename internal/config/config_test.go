package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear any environment variables that might affect the test
	envVars := []string{
		"PORT", "API_KEY", "TOR_SOCKS_ADDR", "TOR_CONTROL_ADDR",
		"MAX_CONCURRENCY", "TIMEOUT", "DB_HOST", "DB_PORT",
		"DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"WEBHOOK_URL", "WEBHOOK_TIMEOUT", "CHECK_INTERVAL",
		"FIRST_BOUNCE_DELAY", "SECOND_BOUNCE_DELAY",
	}

	// Save original values
	originalValues := make(map[string]string)
	for _, key := range envVars {
		if val, exists := os.LookupEnv(key); exists {
			originalValues[key] = val
			os.Unsetenv(key)
		}
	}

	// Restore original values after test
	defer func() {
		for key, val := range originalValues {
			os.Setenv(key, val)
		}
	}()

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("Expected Port '3000', got %s", cfg.Port)
	}

	if cfg.APIKey != "secret-key-change-me" {
		t.Errorf("Expected default API key, got %s", cfg.APIKey)
	}

	if cfg.TorSocksAddr != "tor:9050" {
		t.Errorf("Expected TorSocksAddr 'tor:9050', got %s", cfg.TorSocksAddr)
	}

	if cfg.MaxConcurrency != 5 {
		t.Errorf("Expected MaxConcurrency 5, got %d", cfg.MaxConcurrency)
	}

	if cfg.Timeout != 45*time.Second {
		t.Errorf("Expected Timeout 45s, got %v", cfg.Timeout)
	}

	if cfg.DBHost != "postgres" {
		t.Errorf("Expected DBHost 'postgres', got %s", cfg.DBHost)
	}

	if cfg.DBPort != 5432 {
		t.Errorf("Expected DBPort 5432, got %d", cfg.DBPort)
	}

	if cfg.DBUser != "postgres" {
		t.Errorf("Expected DBUser 'postgres', got %s", cfg.DBUser)
	}

	if cfg.DBName != "verifier" {
		t.Errorf("Expected DBName 'verifier', got %s", cfg.DBName)
	}

	if cfg.DBSSLMode != "disable" {
		t.Errorf("Expected DBSSLMode 'disable', got %s", cfg.DBSSLMode)
	}

	if cfg.CheckInterval != 1*time.Minute {
		t.Errorf("Expected CheckInterval 1m, got %v", cfg.CheckInterval)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set custom environment variables
	testEnvVars := map[string]string{
		"PORT":                "8080",
		"API_KEY":             "custom-api-key",
		"TOR_SOCKS_ADDR":      "127.0.0.1:9050",
		"MAX_CONCURRENCY":     "10",
		"TIMEOUT":             "60s",
		"DB_HOST":             "localhost",
		"DB_PORT":             "5433",
		"DB_USER":             "testuser",
		"DB_PASSWORD":         "testpassword",
		"DB_NAME":             "testdb",
		"DB_SSLMODE":          "require",
		"WEBHOOK_URL":         "https://webhook.test.com",
		"WEBHOOK_TIMEOUT":     "15s",
		"CHECK_INTERVAL":      "30s",
		"FIRST_BOUNCE_DELAY":  "5m",
		"SECOND_BOUNCE_DELAY": "1h",
	}

	// Save original values and set test values
	originalValues := make(map[string]string)
	for key, val := range testEnvVars {
		if origVal, exists := os.LookupEnv(key); exists {
			originalValues[key] = origVal
		}
		os.Setenv(key, val)
	}

	// Restore original values after test
	defer func() {
		for key := range testEnvVars {
			if origVal, exists := originalValues[key]; exists {
				os.Setenv(key, origVal)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Expected Port '8080', got %s", cfg.Port)
	}

	if cfg.APIKey != "custom-api-key" {
		t.Errorf("Expected APIKey 'custom-api-key', got %s", cfg.APIKey)
	}

	if cfg.TorSocksAddr != "127.0.0.1:9050" {
		t.Errorf("Expected TorSocksAddr '127.0.0.1:9050', got %s", cfg.TorSocksAddr)
	}

	if cfg.MaxConcurrency != 10 {
		t.Errorf("Expected MaxConcurrency 10, got %d", cfg.MaxConcurrency)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout 60s, got %v", cfg.Timeout)
	}

	if cfg.DBHost != "localhost" {
		t.Errorf("Expected DBHost 'localhost', got %s", cfg.DBHost)
	}

	if cfg.DBPort != 5433 {
		t.Errorf("Expected DBPort 5433, got %d", cfg.DBPort)
	}

	if cfg.WebhookURL != "https://webhook.test.com" {
		t.Errorf("Expected WebhookURL 'https://webhook.test.com', got %s", cfg.WebhookURL)
	}

	if cfg.WebhookTimeout != 15*time.Second {
		t.Errorf("Expected WebhookTimeout 15s, got %v", cfg.WebhookTimeout)
	}

	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("Expected CheckInterval 30s, got %v", cfg.CheckInterval)
	}

	if cfg.FirstBounceDelay != 5*time.Minute {
		t.Errorf("Expected FirstBounceDelay 5m, got %v", cfg.FirstBounceDelay)
	}

	if cfg.SecondBounceDelay != 1*time.Hour {
		t.Errorf("Expected SecondBounceDelay 1h, got %v", cfg.SecondBounceDelay)
	}
}

func TestResolveDatabaseDSN_FromComponents(t *testing.T) {
	// Clear DATABASE_DSN to test component-based DSN
	originalDSN, hasDSN := os.LookupEnv("DATABASE_DSN")
	if hasDSN {
		os.Unsetenv("DATABASE_DSN")
	}
	defer func() {
		if hasDSN {
			os.Setenv("DATABASE_DSN", originalDSN)
		}
	}()

	cfg := &Config{
		DBUser:     "testuser",
		DBPassword: "testpass",
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "testdb",
		DBSSLMode:  "disable",
	}

	dsn := cfg.ResolveDatabaseDSN()
	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"

	if dsn != expected {
		t.Errorf("Expected DSN %s, got %s", expected, dsn)
	}
}

func TestResolveDatabaseDSN_FromEnvVar(t *testing.T) {
	customDSN := "postgres://custom:secret@custom-host:5433/customdb?sslmode=require"

	os.Setenv("DATABASE_DSN", customDSN)
	defer os.Unsetenv("DATABASE_DSN")

	cfg := &Config{
		DBUser:     "testuser",
		DBPassword: "testpass",
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "testdb",
		DBSSLMode:  "disable",
	}

	dsn := cfg.ResolveDatabaseDSN()

	if dsn != customDSN {
		t.Errorf("Expected DSN from env var %s, got %s", customDSN, dsn)
	}
}

func TestGetEnv(t *testing.T) {
	testKey := "TEST_CONFIG_KEY"
	testValue := "test_value"

	// Test with env var not set
	os.Unsetenv(testKey)
	result := getEnv(testKey, "default")
	if result != "default" {
		t.Errorf("Expected 'default', got %s", result)
	}

	// Test with env var set
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	result = getEnv(testKey, "default")
	if result != testValue {
		t.Errorf("Expected '%s', got %s", testValue, result)
	}
}

func TestGetEnvInt(t *testing.T) {
	testKey := "TEST_CONFIG_INT"

	// Test with env var not set
	os.Unsetenv(testKey)
	result := getEnvInt(testKey, 42)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// Test with valid integer
	os.Setenv(testKey, "100")
	result = getEnvInt(testKey, 42)
	if result != 100 {
		t.Errorf("Expected 100, got %d", result)
	}

	// Test with invalid integer
	os.Setenv(testKey, "not-a-number")
	result = getEnvInt(testKey, 42)
	if result != 42 {
		t.Errorf("Expected default 42 for invalid int, got %d", result)
	}

	os.Unsetenv(testKey)
}

func TestGetEnvDuration(t *testing.T) {
	testKey := "TEST_CONFIG_DURATION"

	// Test with env var not set
	os.Unsetenv(testKey)
	result := getEnvDuration(testKey, 30*time.Second)
	if result != 30*time.Second {
		t.Errorf("Expected 30s, got %v", result)
	}

	// Test with valid duration
	os.Setenv(testKey, "5m")
	result = getEnvDuration(testKey, 30*time.Second)
	if result != 5*time.Minute {
		t.Errorf("Expected 5m, got %v", result)
	}

	// Test with invalid duration
	os.Setenv(testKey, "invalid")
	result = getEnvDuration(testKey, 30*time.Second)
	if result != 30*time.Second {
		t.Errorf("Expected default 30s for invalid duration, got %v", result)
	}

	os.Unsetenv(testKey)
}

func TestConfig_Structure(t *testing.T) {
	cfg := Config{
		Port:              "8080",
		APIKey:            "test-key",
		Host:              "localhost",
		TorSocksAddr:      "127.0.0.1:9050",
		TorControlAddr:    "127.0.0.1:9051",
		MaxConcurrency:    10,
		Timeout:           30 * time.Second,
		DatabaseDSN:       "postgres://user:pass@localhost:5432/db",
		DBHost:            "localhost",
		DBPort:            5432,
		DBUser:            "user",
		DBPassword:        "pass",
		DBName:            "db",
		DBSSLMode:         "disable",
		WebhookURL:        "https://webhook.com",
		WebhookTimeout:    10 * time.Second,
		CheckInterval:     1 * time.Minute,
		FirstBounceDelay:  2 * time.Minute,
		SecondBounceDelay: 6 * time.Hour,
	}

	// Verify all fields are accessible
	if cfg.Port != "8080" {
		t.Errorf("Expected Port '8080', got %s", cfg.Port)
	}

	if cfg.MaxConcurrency != 10 {
		t.Errorf("Expected MaxConcurrency 10, got %d", cfg.MaxConcurrency)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout 30s, got %v", cfg.Timeout)
	}

	if cfg.FirstBounceDelay != 2*time.Minute {
		t.Errorf("Expected FirstBounceDelay 2m, got %v", cfg.FirstBounceDelay)
	}

	if cfg.SecondBounceDelay != 6*time.Hour {
		t.Errorf("Expected SecondBounceDelay 6h, got %v", cfg.SecondBounceDelay)
	}
}
