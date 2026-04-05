package service

import (
	"fmt"
	"log"

	"gopkg.in/telebot.v3"

	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

// Bot представляет Telegram-бота и его зависимости.
type Bot struct {
	Telebot           *telebot.Bot
	GigaChat          *gigachat.GigaChatClient
	SaluteSpeech      *salutespeech.SaluteSpeechClient
	DB                *repository.DB
	UserRepo          *repository.UserRepository
	MeetingRepo       *repository.MeetingRepository
	TranscriptionRepo *repository.TranscriptionRepository
	SummaryRepo       *repository.SummaryRepository
	AudioStoragePath  string
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
		Telebot:           bot,
		GigaChat:          gigaChat,
		SaluteSpeech:      saluteSpeech,
		DB:                db,
		UserRepo:          repository.NewUserRepository(db),
		MeetingRepo:       repository.NewMeetingRepository(db),
		TranscriptionRepo: repository.NewTranscriptionRepository(db),
		SummaryRepo:       repository.NewSummaryRepository(db),
		AudioStoragePath:  audioStoragePath,
	}
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
