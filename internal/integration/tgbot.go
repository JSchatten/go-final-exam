package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/JSchatten/go-final-exam/internal/config"
	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

// Bot represents the Telegram bot
type Bot struct {
	Telebot          *telebot.Bot
	Config           *config.Config
	GigaChat         *gigachat.GigaChatClient
	DB               *repository.DB
	AudioStoragePath string
}

// NewBot creates a new Telegram bot instance
func NewBot(cfg *config.Config, gigaChat *gigachat.GigaChatClient, db *repository.DB) (*Bot, error) {
	// Create bot settings
	settings := telebot.Settings{
		// URL:    "", // Use default Telegram URL
		Token:  cfg.TelegramToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	// Create new bot
	bot, err := telebot.NewBot(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return &Bot{
		Telebot:          bot,
		Config:           cfg,
		GigaChat:         gigaChat,
		DB:               db,
		AudioStoragePath: cfg.AudioStoragePath,
	}, nil
}

// Start starts the bot and sets up handlers
func (b *Bot) Start() {
	// Set up command handlers
	b.Telebot.Handle("/start", b.handleStart)
	b.Telebot.Handle("/list", b.handleList)
	b.Telebot.Handle("/get", b.handleGet)
	b.Telebot.Handle("/find", b.handleFind)
	b.Telebot.Handle("/chat", b.handleChat)

	// Set up voice message handler
	b.Telebot.Handle(telebot.OnVoice, b.handleVoice)

	// Set up audio file handler
	b.Telebot.Handle(telebot.OnAudio, b.handleAudio)

	// Set up text message handler for continuing chat
	b.Telebot.Handle(telebot.OnText, b.handleText)

	// Start the bot
	log.Println("Bot is starting...")
	b.Telebot.Start()
}

// Stop stops the bot
func (b *Bot) Stop() {
	b.Telebot.Stop()
	log.Println("Bot stopped")
}

// handleStart handles /start command
func (b *Bot) handleStart(c telebot.Context) error {
	user := c.Sender()
	// Здесь будет логика регистрации пользователя

	fmt.Println(user.ID)
	return c.Send(fmt.Sprintf("Привет, %s! Ты успешно зарегистрирован.", user.FirstName))
}

// handleList handles /list command
func (b *Bot) handleList(c telebot.Context) error {
	// TODO: Implement listing saved meetings from database
	return c.Send("Список встреч: (пока не реализовано)")

}

// handleGet handles /get command
func (b *Bot) handleGet(c telebot.Context) error {
	// TODO: Implement getting meeting transcript by ID from database
	return c.Send("Получение текста встречи: (пока не реализовано)")

}

// handleFind handles /find command
func (b *Bot) handleFind(c telebot.Context) error {
	query := c.Text()[6:] // /find query -> извлекаем запрос
	if strings.Trim(query, " ") == "" {
		return c.Send("Укажите ключевые слова для поиска. Пример: /find проект")
	}
	// TODO: Поиск встреч по ключевым словам
	return c.Send(fmt.Sprintf("Ищу встречи по запросу: %q (пока не реализовано)", query))
}

// handleChat handles /chat command
func (b *Bot) handleChat(c telebot.Context) error {
	prompt := c.Text()[6:] // /chat hello -> извлекаем запрос
	if prompt == "" {
		return c.Send("Напишите запрос после команды. Пример: /chat Как дела?")
	}
	// TODO: Отправить запрос в GigaChat и получить ответ
	return c.Send(fmt.Sprintf("Запрос к GigaChat: %q (пока не реализовано)", prompt))
}

// handleVoice handles voice messages
func (b *Bot) handleVoice(c telebot.Context) error {
	voice := c.Message().Voice
	user := c.Sender()

	log.Printf("Получено голосовое сообщение: %d сек, FileID: %s", voice.Duration, voice.FileID)

	// 1. Получаем информацию о файле (включая FilePath)
	file, err := b.Telebot.FileByID(voice.FileID)
	if err != nil {
		log.Printf("Не удалось получить информацию о файле: %v", err)
		return c.Send("Не удалось загрузить аудиофайл.")
	}

	log.Printf("Mime types %s", voice.MIME)

	// 2. Формируем путь для сохранения
	audioDir := b.AudioStoragePath
	audioPath := filepath.Join(audioDir, fmt.Sprintf("voice_%d_%d.ogg", user.ID, time.Now().Unix()))

	// 3. Создаём директорию, если не существует
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		log.Printf("Не удалось создать папку %s: %v", audioDir, err)
		return c.Send("Ошибка при сохранении аудио.")
	}

	// 4. Открываем выходной файл
	outFile, err := os.Create(audioPath)
	if err != nil {
		log.Printf("Не удалось создать файл %s: %v", audioPath, err)
		return c.Send("Ошибка при сохранении аудио.")
	}
	defer outFile.Close()

	// 5. Формируем URL для скачивания
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Config.TelegramToken, file.FilePath)

	// 6. Скачиваем
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Ошибка при скачивании файла: %v", err)
		return c.Send("Не удалось скачать аудио.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP ошибка при скачивании: %d", resp.StatusCode)
		return c.Send("Не удалось скачать аудио.")
	}

	// 7. Копируем содержимое
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("Ошибка при записи файла: %v", err)
		return c.Send("Ошибка при сохранении аудио.")
	}

	log.Printf("Голосовое сообщение успешно сохранено: %s", audioPath)

	// // 8. Теперь можно отправить в SaluteSpeech
	// text, err := recognizeSpeechWithSalute(cfg, audioPath)
	// if err != nil {
	// 	log.Printf("Ошибка распознавания речи: %v", err)
	// 	return c.Send("Не удалось распознать речь.")
	// }

	return c.Send("Голосовое сообщение получено. Обрабатываю... (распознавание речи пока не реализовано)")
}

// handleAudio handles audio files
func (b *Bot) handleAudio(c telebot.Context) error {
	transcript := "[Транскрипция аудиофайла будет здесь]"

	// Generate summary using GigaChat
	summaryPrompt := fmt.Sprintf("Сделай краткую выжимку из следующего текста:\n\n%s", transcript)
	summary, err := b.GigaChat.SendMessage(summaryPrompt)
	if err != nil {
		log.Printf("Failed to get summary from GigaChat: %v", err)
		summary = "Не удалось сгенерировать краткую выжимку."
	}

	// Send results to user
	result := fmt.Sprintf("Транскрипция:\n%s\n\nКраткая выжимка:\n%s", transcript, summary)
	return c.Send(result)
}

// handleText handles text messages (for continuing chat without /chat command)
func (b *Bot) handleText(c telebot.Context) error {
	text := c.Text()
	// Можно использовать для накопления контекста, анализа, etc.
	return c.Send(fmt.Sprintf("Вы написали: %q (текст пока не обрабатывается)", text))
}

// RunBot runs the Telegram bot with context for graceful shutdown
func RunBot(ctx context.Context, cfg *config.Config, gigaChat *gigachat.GigaChatClient, db *repository.DB) {
	bot, err := NewBot(cfg, gigaChat, db)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the bot in a separate goroutine
	go func() {
		bot.Start()
	}()

	// Wait for context cancellation
	<-ctx.Done()

	log.Println("Shutting down bot...")
	bot.Stop()
}
