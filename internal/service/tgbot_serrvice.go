package service

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

// Bot представляет Telegram-бота и его зависимости.
type Bot struct {
	Telebot          *telebot.Bot
	GigaChat         *gigachat.GigaChatClient
	SaluteSpeech     *salutespeech.SaluteSpeechClient
	DB               *repository.DB
	AudioStoragePath string
}

// NewBot создаёт новый экземпляр бота с готовыми зависимостями
func NewBot(
	bot *telebot.Bot,
	gigaChat *gigachat.GigaChatClient,
	saluteSpeech *salutespeech.SaluteSpeechClient,
	db *repository.DB,
	audioStoragePath string,
) *Bot {
	return &Bot{
		Telebot:          bot,
		GigaChat:         gigaChat,
		SaluteSpeech:     saluteSpeech,
		DB:               db,
		AudioStoragePath: audioStoragePath,
	}
}

// HandleStart обрабатывает команду /start.
func (b *Bot) HandleStart(c telebot.Context) error {
	user := c.Sender()
	return c.Send(fmt.Sprintf("Привет, %s! Ты успешно зарегистрирован.", user.FirstName))
}

// HandleList обрабатывает команду /list.
func (b *Bot) HandleList(c telebot.Context) error {
	return c.Send("Список встреч: (пока не реализовано)")
}

// HandleGet обрабатывает команду /get.
func (b *Bot) HandleGet(c telebot.Context) error {
	return c.Send("Получение текста встречи: (пока не реализовано)")
}

// HandleFind обрабатывает команду /find.
func (b *Bot) HandleFind(c telebot.Context) error {
	query := c.Text()[6:]
	if strings.Trim(query, " ") == "" {
		return c.Send("Укажите ключевые слова для поиска. Пример: /find проект")
	}
	return c.Send(fmt.Sprintf("Ищу встречи по запросу: %q (пока не реализовано)", query))
}

// HandleChat обрабатывает команду /chat.
func (b *Bot) HandleChat(c telebot.Context) error {
	prompt := c.Text()[6:]
	if prompt == "" {
		return c.Send("Напишите запрос после команды. Пример: /chat Как дела?")
	}
	response, err := b.GigaChat.SendMessage(prompt)
	if err != nil {
		log.Printf("Failed to get response from GigaChat: %v", err)
		return c.Send("Не удалось получить ответ от GigaChat.")
	}
	return c.Send(response)
}

// HandleVoice обрабатывает голосовые сообщения.
func (b *Bot) HandleVoice(c telebot.Context) error {
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

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Telebot.Token, file.FilePath)

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

// HandleAudio обрабатывает аудиофайлы.
func (b *Bot) HandleAudio(c telebot.Context) error {
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

// HandleText обрабатывает текстовые сообщения.
func (b *Bot) HandleText(c telebot.Context) error {
	text := c.Text()
	return c.Send(fmt.Sprintf("Вы написали: %q (текст пока не обрабатывается)", text))
}
