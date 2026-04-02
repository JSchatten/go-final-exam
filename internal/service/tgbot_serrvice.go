package service

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

// Bot представляет Telegram-бота и его зависимости.
type Bot struct {
	Telebot          *telebot.Bot
	Config           *config.Config
	GigaChat         *gigachat.GigaChatClient
	DB               *repository.DB
	AudioStoragePath string
}

// NewBot создаёт новый экземпляр бота.
func NewBot(cfg *config.Config, gigaChat *gigachat.GigaChatClient, db *repository.DB) (*Bot, error) {
	settings := telebot.Settings{
		Token:  cfg.TelegramToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

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

// Start запускает бота и настраивает обработчики команд и сообщений.
func (b *Bot) Start() {
	b.Telebot.Handle("/start", b.handleStart)
	b.Telebot.Handle("/list", b.handleList)
	b.Telebot.Handle("/get", b.handleGet)
	b.Telebot.Handle("/find", b.handleFind)
	b.Telebot.Handle("/chat", b.handleChat)
	b.Telebot.Handle(telebot.OnVoice, b.handleVoice)
	b.Telebot.Handle(telebot.OnAudio, b.handleAudio)
	b.Telebot.Handle(telebot.OnText, b.handleText)

	log.Println("Bot is starting...")
	b.Telebot.Start()
}

// Stop останавливает бота.
func (b *Bot) Stop() {
	b.Telebot.Stop()
	log.Println("Bot stopped")
}

// handleStart обрабатывает команду /start.
func (b *Bot) handleStart(c telebot.Context) error {
	user := c.Sender()
	fmt.Println(user.ID)
	return c.Send(fmt.Sprintf("Привет, %s! Ты успешно зарегистрирован.", user.FirstName))
}

// handleList обрабатывает команду /list.
func (b *Bot) handleList(c telebot.Context) error {
	return c.Send("Список встреч: (пока не реализовано)")
}

// handleGet обрабатывает команду /get.
func (b *Bot) handleGet(c telebot.Context) error {
	return c.Send("Получение текста встречи: (пока не реализовано)")
}

// handleFind обрабатывает команду /find.
func (b *Bot) handleFind(c telebot.Context) error {
	query := c.Text()[6:]
	if strings.Trim(query, " ") == "" {
		return c.Send("Укажите ключевые слова для поиска. Пример: /find проект")
	}
	return c.Send(fmt.Sprintf("Ищу встречи по запросу: %q (пока не реализовано)", query))
}

// handleChat обрабатывает команду /chat.
func (b *Bot) handleChat(c telebot.Context) error {
	prompt := c.Text()[6:]
	if prompt == "" {
		return c.Send("Напишите запрос после команды. Пример: /chat Как дела?")
	}
	return c.Send(fmt.Sprintf("Запрос к GigaChat: %q (пока не реализовано)", prompt))
}

// handleVoice обрабатывает голосовые сообщения.
func (b *Bot) handleVoice(c telebot.Context) error {
	voice := c.Message().Voice
	user := c.Sender()

	log.Printf("Получено голосовое сообщение: %d сек, FileID: %s", voice.Duration, voice.FileID)

	file, err := b.Telebot.FileByID(voice.FileID)
	if err != nil {
		log.Printf("Не удалось получить информацию о файле: %v", err)
		return c.Send("Не удалось загрузить аудиофайл.")
	}

	audioDir := b.AudioStoragePath
	audioPath := filepath.Join(audioDir, fmt.Sprintf("voice_%d_%d.ogg", user.ID, time.Now().Unix()))

	if err := os.MkdirAll(audioDir, 0755); err != nil {
		log.Printf("Не удалось создать папку %s: %v", audioDir, err)
		return c.Send("Ошибка при сохранении аудио.")
	}

	outFile, err := os.Create(audioPath)
	if err != nil {
		log.Printf("Не удалось создать файл %s: %v", audioPath, err)
		return c.Send("Ошибка при сохранении аудио.")
	}
	defer outFile.Close()

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Config.TelegramToken, file.FilePath)

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

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("Ошибка при записи файла: %v", err)
		return c.Send("Ошибка при сохранении аудио.")
	}

	log.Printf("Голосовое сообщение успешно сохранено: %s", audioPath)

	return c.Send("Голосовое сообщение получено. Обрабатываю... (распознавание речи пока не реализовано)")
}

// handleAudio обрабатывает аудиофайлы.
func (b *Bot) handleAudio(c telebot.Context) error {
	transcript := "[Транскрипция аудиофайла будет здесь]"

	summaryPrompt := fmt.Sprintf("Сделай краткую выжимку из следующего текста:\n\n%s", transcript)
	summary, err := b.GigaChat.SendMessage(summaryPrompt)
	if err != nil {
		log.Printf("Failed to get summary from GigaChat: %v", err)
		summary = "Не удалось сгенерировать краткую выжимку."
	}

	result := fmt.Sprintf("Транскрипция:\n%s\n\nКраткая выжимка:\n%s", transcript, summary)
	return c.Send(result)
}

// handleText обрабатывает текстовые сообщения.
func (b *Bot) handleText(c telebot.Context) error {
	text := c.Text()
	return c.Send(fmt.Sprintf("Вы написали: %q (текст пока не обрабатывается)", text))
}

// RunBot запускает бота с поддержкой graceful shutdown через контекст.
func RunBot(ctx context.Context, cfg *config.Config, gigaChat *gigachat.GigaChatClient, db *repository.DB) {
	bot, err := NewBot(cfg, gigaChat, db)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	go func() {
		bot.Start()
	}()

	<-ctx.Done()

	log.Println("Shutting down bot...")
	bot.Stop()
}
