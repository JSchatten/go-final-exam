// cmd/main/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/telebot.v3"

	"github.com/JSchatten/go-final-exam/internal/config"
	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/logger"
	"github.com/JSchatten/go-final-exam/internal/repository"
	"github.com/JSchatten/go-final-exam/internal/service"
	"github.com/JSchatten/go-final-exam/internal/worker"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Get().Fatal().Msgf("Failed to load config: %v", err)
	}
	logger.SetLevel(*cfg.Loglevel)

	// Выводим конфигурацию (для отладки)
	logger.Get().Info().Msg("Configuration loaded")
	logger.Get().Debug().Msgf("Telegram Token: %s", cfg.TelegramToken)
	logger.Get().Debug().Msgf("GigaChat ClientID: %s", cfg.Gigachat.ClientID)
	logger.Get().Debug().Msgf("GigaChat Scope: %s", cfg.Gigachat.Scope)
	logger.Get().Debug().Msgf("SaluteSpeech ClientID: %s", cfg.SaluteSpeech.ClientID)
	logger.Get().Debug().Msgf("SaluteSpeech Scope: %s", cfg.SaluteSpeech.Scope)
	logger.Get().Debug().Msgf("Database: %s@%s:%d/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	logger.Get().Debug().Msgf("Audio Storage Path: %s", cfg.AudioStoragePath)

	// Создаём Telebot
	settings := telebot.Settings{
		Token: cfg.TelegramToken,
		Poller: &telebot.LongPoller{
			Timeout:        10 * time.Second,
			AllowedUpdates: []string{"message", "callback_query"},
		},
	}

	telebotInstance, err := telebot.NewBot(settings)
	if err != nil {
		logger.Get().Fatal().Msgf("Failed to create Telegram bot: %v", err)
	}

	// GigaChat Client
	gigaChat, err := gigachat.NewGigaChatClient(cfg.Gigachat)
	if err != nil {
		logger.Get().Fatal().Msgf("Failed to create GigaChat client: %v", err)
	}

	// SaluteSpeech Client
	saluteSpeech, err := salutespeech.NewSaluteSpeechClient(cfg.SaluteSpeech)
	if err != nil {
		logger.Get().Fatal().Msgf("Failed to create SaluteSpeech client: %v", err)
	}

	// Database
	db, err := repository.InitDB(cfg.Database)
	if err != nil {
		logger.Get().Fatal().Msgf("Failed to init DB: %v", err)
	}

	// Создаём экземпляр бота-сервиса
	bot := service.NewBotService(
		telebotInstance,
		gigaChat,
		saluteSpeech,
		db,
		cfg.AudioStoragePath,
	)

	// === Настраиваем маршруты (как в Gin) ===

	// telebotInstance.Handle("/start", bot.HandleStart)
	// https://deepwiki.com/go-telebot/telebot/6.1-reply-markup
	telebotInstance.Handle(&service.BtnStart, bot.HandleStart)
	// telebotInstance.Handle(*bot.MenuTest, bot.HandleTest)

	telebotInstance.Handle("/list", bot.HandleList)
	telebotInstance.Handle("/get", bot.HandleGet)
	telebotInstance.Handle("/find", bot.HandleFind)
	telebotInstance.Handle("/chat", bot.HandleChat)
	telebotInstance.Handle(telebot.OnVoice, bot.HandleVoice)
	telebotInstance.Handle(telebot.OnAudio, bot.HandleAudio)
	telebotInstance.Handle(telebot.OnText, bot.HandleText)

	logger.Get().Info().Msg("Application is running...")

	// Создаём контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Перехватываем системные сигналы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		logger.Get().Info().Msgf("Received signal: %s. Shutting down...", sig)
		cancel()
	}()

	// Запускаем errgroup
	g, gCtx := errgroup.WithContext(ctx)

	// Запуск бота с graceful shutdown
	g.Go(func() error {
		logger.Get().Info().Msg("Bot: starting...")

		// Запускаем Start в отдельной горутине
		go bot.Telebot.Start()

		// Ждём сигнала остановки
		<-gCtx.Done()

		logger.Get().Info().Msg("Bot: stopping...")
		bot.Telebot.Stop() // ← Это заставит Start() вернуться

		return nil
	})

	// Запуск воркера
	transcriptionWorker := worker.NewTranscriptionWorker(
		bot,
	)
	g.Go(func() error {
		return transcriptionWorker.Start(gCtx)
	})

	// Ждём завершения всех задач
	if err := g.Wait(); err != nil && err != ctx.Err() {
		logger.Get().Fatal().Msgf("Application error: %v", err)
	}

	logger.Get().Info().Msg("Application stopped.")

}
