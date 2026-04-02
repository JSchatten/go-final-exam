// cmd/main/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JSchatten/go-final-exam/internal/config"
	"github.com/JSchatten/go-final-exam/internal/repository"
	"github.com/JSchatten/go-final-exam/internal/service"
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

	// Выводим конфигурацию (для отладки)
	fmt.Println("Configuration loaded:")
	fmt.Printf("Telegram Token: %s\n", cfg.TelegramToken)
	fmt.Printf("GigaChat ClientID: %s\n", cfg.GigaChatClientID)
	fmt.Printf("Sber SaluteSpeech ClientID: %s\n", cfg.SaluteSpeechClientID)
	fmt.Printf("Database: %s@%s:%d/%s\n", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	fmt.Printf("Files path: %s\n", cfg.AudioStoragePath)

	fmt.Println("Application is running...")

	// Создаём контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Перехватываем системные сигналы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-quit
		log.Printf("Received signal: %s. Shutting down...\n", sig)
		cancel()
	}()

	// Запускаем бота через сервис
	service.RunBot(ctx, cfg, nil, db)

	// После завершения RunBot (когда контекст отменён) — приложение корректно остановится
	log.Println("Application stopped.")
}
