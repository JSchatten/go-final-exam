// internal/config/config.go
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

// Config holds the application configuration
type Config struct {
	// Telegram
	TelegramToken string `json:"telegram_token"`

	// Modules
	Database     *repository.Config   `json:"Database"`
	SaluteSpeech *salutespeech.Config `json:"SaluteSpeech"`
	Gigachat     *gigachat.Config     `json:"Gigachat"`

	// Files
	AudioStoragePath string `json:"audio_storage_path"` // путь к папке для сохранения аудиофайлов

}

func LoadConfig() (*Config, error) {
	configFile := flag.String("config", "settings.json", "path to config file")
	flag.Parse()

	var cfg Config

	// Загружаем из JSON-файла
	file, err := os.Open(*configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", *configFile, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields() // строгая проверка полей

	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Инициализируем подконфиги, если не заданы в JSON
	if cfg.Gigachat == nil {
		cfg.Gigachat = &gigachat.Config{}
	}
	if cfg.SaluteSpeech == nil {
		cfg.SaluteSpeech = &salutespeech.Config{}
	}
	if cfg.Database == nil {
		cfg.Database = &repository.Config{}
	}

	// Переопределение через переменные окружения

	// Telegram
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.TelegramToken = token
	}

	// Audio storage path
	if path := os.Getenv("AUDIO_STORAGE_PATH"); path != "" {
		cfg.AudioStoragePath = path
	}

	// GigaChat
	if v := os.Getenv("GIGACHAT_CLIENT_ID"); v != "" {
		cfg.Gigachat.ClientID = v
	}
	if v := os.Getenv("GIGACHAT_SCOPE"); v != "" {
		cfg.Gigachat.Scope = v
	}
	if v := os.Getenv("GIGACHAT_AUTH_KEY"); v != "" {
		cfg.Gigachat.AuthKey = v
	}

	// SaluteSpeech
	if v := os.Getenv("SALUTESPEECH_CLIENT_ID"); v != "" {
		cfg.SaluteSpeech.ClientID = v
	}
	if v := os.Getenv("SALUTESPEECH_SCOPE"); v != "" {
		cfg.SaluteSpeech.Scope = v
	}
	if v := os.Getenv("SALUTESPEECH_AUTH_KEY"); v != "" {
		cfg.SaluteSpeech.AuthKey = v
	}

	// Database
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		} else {
			return nil, fmt.Errorf("invalid DB_PORT: %s", v)
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.Name = v
	}

	// Валидация обязательных полей
	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("required config: TELEGRAM_TOKEN is missing")
	}
	if cfg.Gigachat.ClientID == "" || cfg.Gigachat.AuthKey == "" {
		return nil, fmt.Errorf("required config: GIGACHAT_CLIENT_ID and GIGACHAT_AUTH_KEY are required")
	}
	if cfg.SaluteSpeech.ClientID == "" || cfg.SaluteSpeech.AuthKey == "" {
		return nil, fmt.Errorf("required config: SALUTESPEECH_CLIENT_ID and SALUTESPEECH_AUTH_KEY are required")
	}
	if cfg.Database.Host == "" || cfg.Database.Port == 0 || cfg.Database.User == "" || cfg.Database.Name == "" {
		return nil, fmt.Errorf("required config: database connection parameters are incomplete")
	}
	if cfg.AudioStoragePath == "" {
		return nil, fmt.Errorf("required config: AUDIO_STORAGE_PATH is missing")
	}

	return &cfg, nil
}
