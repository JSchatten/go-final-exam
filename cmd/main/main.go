// cmd/main/main.go
package main

import (
	"fmt"
	"log"

	"github.com/JSchatten/go-final-exam/internal/config"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Формируем DSN для PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	// Инициализируем БД
	db, err := repository.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Выводим только что загруженные токены (безопасно - в продакшене убрать)
	fmt.Println("Configuration loaded:")
	fmt.Printf("Telegram Token: %s\n", cfg.TelegramToken)
	fmt.Printf("GigaChat GigaChatClientID: %s\n", cfg.GigaChatClientID)
	fmt.Printf("Sber SaluteSpeechClientID: %s\n", cfg.SaluteSpeechClientID)
	fmt.Printf("Database: %s@%s:%d/%s\n", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	fmt.Println("Hello, Go! Application is running...")
	// Здесь будет запуск бота и других сервисов
}
