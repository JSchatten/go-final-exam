package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	TelegramToken string `json:"telegram_token"`
	GigaChatToken string `json:"gigachat_token"`
	SberAuthKey   string `json:"sber_auth_key"`
	DBHost        string `json:"db_host"`
	DBPort        int    `json:"db_port"`
	DBUser        string `json:"db_user"`
	DBPassword    string `json:"db_password"`
	DBName        string `json:"db_name"`
}

// LoadConfig loads configuration from file and command-line flags
func LoadConfig() (*Config, error) {
	configFile := flag.String("config", "config.json", "path to config file")
	flag.Parse()

	// Read config file
	file, err := os.Open(*configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Override with environment variables if present
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.TelegramToken = token
	}
	if token := os.Getenv("GIGACHAT_TOKEN"); token != "" {
		cfg.GigaChatToken = token
	}
	if key := os.Getenv("SBER_AUTH_KEY"); key != "" {
		cfg.SberAuthKey = key
	}
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.DBHost = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		// ignore error, keep default
		if p, err := fmt.Sscanf(port, "%d", &cfg.DBPort); err != nil || p != 1 {
			return nil, fmt.Errorf("invalid DB_PORT: %s", port)
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.DBUser = user
	}
	if pass := os.Getenv("DB_PASSWORD"); pass != "" {
		cfg.DBPassword = pass
	}
	if name := os.Getenv("DB_NAME"); name != "" {
		cfg.DBName = name
	}

	return &cfg, nil
}
