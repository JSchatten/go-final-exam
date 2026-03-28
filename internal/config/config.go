// internal/config/config.go
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	// Telegram
	TelegramToken string `json:"telegram_token"`

	// GigaChat API
	GigaChatClientID string `json:"gigachat_client_id"`
	GigaChatScope    string `json:"gigachat_scope"`
	GigaChatAuthKey  string `json:"gigachat_auth_key"`

	// SaluteSpeech API
	SaluteSpeechClientID string `json:"salutespeech_client_id"`
	SaluteSpeechScope    string `json:"salutespeech_scope"`
	SaluteSpeechAuthKey  string `json:"salutespeech_auth_key"`

	// Database
	DBHost     string `json:"db_host"`
	DBPort     int    `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
}

// LoadConfig loads configuration from file and command-line flags
func LoadConfig() (*Config, error) {
	configFile := flag.String("config", "settings.json", "path to config file")
	flag.Parse()

	var cfg Config

	// Попробуем открыть файл конфигурации
	file, err := os.Open(*configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Переопределение через переменные окружения (если заданы)

	// === Telegram ===
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.TelegramToken = token
	}

	// === GigaChat ===
	if v := os.Getenv("GIGACHAT_CLIENT_ID"); v != "" {
		cfg.GigaChatClientID = v
	}
	if v := os.Getenv("GIGACHAT_SCOPE"); v != "" {
		cfg.GigaChatScope = v
	}
	if v := os.Getenv("GIGACHAT_AUTHORIZATION_KEY"); v != "" {
		cfg.GigaChatAuthKey = v
	}

	// === SaluteSpeech ===
	if v := os.Getenv("SALUTESPEECH_CLIENT_ID"); v != "" {
		cfg.SaluteSpeechClientID = v
	}
	if v := os.Getenv("SALUTESPEECH_SCOPE"); v != "" {
		cfg.SaluteSpeechScope = v
	}
	if v := os.Getenv("SALUTESPEECH_AUTHORIZATION_KEY"); v != "" {
		cfg.SaluteSpeechAuthKey = v
	}

	// === Database ===
	if v := os.Getenv("DBHOST"); v != "" {
		cfg.DBHost = v
	}
	if v := os.Getenv("DBPORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.DBPort = port
		} else {
			return nil, fmt.Errorf("invalid DBPORT: %s", v)
		}
	}
	if v := os.Getenv("DBUSER"); v != "" {
		cfg.DBUser = v
	}
	if v := os.Getenv("DBPASSWORD"); v != "" {
		cfg.DBPassword = v
	}
	if v := os.Getenv("DBNAME"); v != "" {
		cfg.DBName = v
	}

	return &cfg, nil
}
