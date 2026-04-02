// cmd/main/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/JSchatten/go-final-exam/internal/config"
	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/repository"
	"github.com/JSchatten/go-final-exam/internal/service"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Выводим конфигурацию (для отладки)
	fmt.Println("Configuration loaded:")
	fmt.Printf("Telegram Token: %s\n", cfg.TelegramToken)
	fmt.Printf("GigaChat ClientID: %s\n", cfg.Gigachat.ClientID)
	fmt.Printf("GigaChat Scope: %s\n", cfg.Gigachat.Scope)
	fmt.Printf("SaluteSpeech ClientID: %s\n", cfg.SaluteSpeech.ClientID)
	fmt.Printf("SaluteSpeech Scope: %s\n", cfg.SaluteSpeech.Scope)
	fmt.Printf("Database: %s@%s:%d/%s\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	fmt.Printf("Audio Storage Path: %s\n", cfg.AudioStoragePath)

	// Создаём Telebot
	settings := telebot.Settings{
		Token:  cfg.TelegramToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	telebotInstance, err := telebot.NewBot(settings)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	// GigaChat Client
	gigaChat, err := gigachat.NewGigaChatClient(cfg.Gigachat)
	if err != nil {
		log.Fatalf("Failed to create GigaChat client: %v", err)
	}

	// SaluteSpeech Client
	saluteSpeech, err := salutespeech.NewSaluteSpeechClient(cfg.SaluteSpeech)
	if err != nil {
		log.Fatalf("Failed to create SaluteSpeech client: %v", err)
	}

	// Database
	db, err := repository.InitDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	// Создаём экземпляр бота-сервиса
	bot := service.NewBot(
		telebotInstance,
		gigaChat,
		saluteSpeech,
		db,
		cfg.AudioStoragePath,
	)

	// === Настраиваем маршруты (как в Gin) ===

	telebotInstance.Handle("/start", bot.HandleStart)
	telebotInstance.Handle("/list", bot.HandleList)
	telebotInstance.Handle("/get", bot.HandleGet)
	telebotInstance.Handle("/find", bot.HandleFind)
	telebotInstance.Handle("/chat", bot.HandleChat)
	telebotInstance.Handle(telebot.OnVoice, bot.HandleVoice)
	telebotInstance.Handle(telebot.OnAudio, bot.HandleAudio)
	telebotInstance.Handle(telebot.OnText, bot.HandleText)

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

	// Запускаем бота
	log.Println("Bot is starting...")
	go telebotInstance.Start()

	// Ждём сигнала завершения
	<-ctx.Done()

	log.Println("Shutting down bot...")
	telebotInstance.Stop()
	log.Println("Application stopped.")
}
