package config


import (
	"os"
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
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "3000"),
		APIKey:         getEnv("API_KEY", "secret-key-change-me"),
		TorSocksAddr:   getEnv("TOR_SOCKS_ADDR", "tor:9050"), // 'tor' is docker service name
		TorControlAddr: getEnv("TOR_CONTROL_ADDR", ""),       // e.g., "tor:9051"
		MaxConcurrency: 5, // Tor is slow, keep this low
		Timeout:        45 * time.Second,
	}
}


func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}