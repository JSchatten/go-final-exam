package service

import (
	"context"
	"fmt"

	"strings"

	"gopkg.in/telebot.v3"

	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/logger"
	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/JSchatten/go-final-exam/internal/repository"
	"github.com/rs/zerolog"
)

const (
	MaxSizeMb   = 500
	MaxFileSize = MaxSizeMb * 1024 * 1024 // 500 МБ
)

var AllowedAudioExtensions = map[string]bool{
	".ogg": true,
	".oga": true,
	".mp3": true,
	".wav": true,
}

var (
	MenuInBot = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// Paginator = &telebot.ReplyMarkup{}
	Selector = &telebot.ReplyMarkup{}

	BtnStart = MenuInBot.Text("start")
	btnHelp  = MenuInBot.Text("help")
	btnFind  = MenuInBot.Text("Find")

	// Inline buttons.
	//
	// Pressing it will cause the client to
	// send the bot a callback.
	//
	// Make sure Unique stays unique as per button kind
	// since it's required for callback routing to work.
	//

)

// BotService представляет Telegram-бота и его зависимости.
type BotService struct {
	Logger            zerolog.Logger
	Telebot           *telebot.Bot
	GigaChat          *gigachat.GigaChatClient
	SaluteSpeech      *salutespeech.SaluteSpeechClient
	DB                *repository.DB
	UserRepo          *repository.UserRepository
	MeetingRepo       *repository.MeetingRepository
	TranscriptionRepo *repository.TranscriptionRepository
	SummaryRepo       *repository.SummaryRepository
	ChatHistoryRepo   *repository.ChatHistoryRepository
	AudioStoragePath  string
	MenuTest          *telebot.ReplyMarkup
}

// NewBotService создаёт новый экземпляр бота с готовыми зависимостями
func NewBotService(
	bot *telebot.Bot,
	gigaChat *gigachat.GigaChatClient,
	saluteSpeech *salutespeech.SaluteSpeechClient,
	db *repository.DB,
	audioStoragePath string,
) *BotService {

	MenuTest := &telebot.ReplyMarkup{ResizeKeyboard: true}

	btnTestInfo := MenuInBot.Text("TestInfo")

	bs := &BotService{
		Logger:            logger.WithContext(context.Background()).With().Str("component", "BotService").Logger(),
		Telebot:           bot,
		GigaChat:          gigaChat,
		SaluteSpeech:      saluteSpeech,
		DB:                db,
		UserRepo:          repository.NewUserRepository(db),
		MeetingRepo:       repository.NewMeetingRepository(db),
		TranscriptionRepo: repository.NewTranscriptionRepository(db),
		SummaryRepo:       repository.NewSummaryRepository(db),
		ChatHistoryRepo:   repository.NewChatHistoryRepository(db),
		AudioStoragePath:  audioStoragePath,
		MenuTest:          MenuTest,
	}

	bs.Telebot.Handle(&btnTestInfo, func(c telebot.Context) error {
		return c.Reply("TestInfo", MenuTest)
	})

	return bs
}

// getCtx безопасно извлекает контекст из telebot.Context или возвращает background
func (b *BotService) getCtx(c telebot.Context) context.Context {
	if ctx, ok := c.Get("ctx").(context.Context); ok {
		return ctx
	}
	return context.Background()
}

// HandleChat обрабатывает команду /chat.
func (b *BotService) HandleChat(c telebot.Context) error {
	prompt := c.Message().Payload // strings.TrimSpace(c.Text()[5:])
	if prompt == "" {
		return c.Reply("Напишите запрос после команды. Пример: /chat Как дела?")
	}

	response, err := b.GigaChat.SendChatMessage(prompt)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to get response from GigaChat: %v", err)
		return c.Reply("Не удалось получить ответ от GigaChat.")
	}

	// Сохраняем в историю
	ctx := b.getCtx(c)
	err = b.ChatHistoryRepo.Create(ctx, c.Sender().ID, prompt, response)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to save chat history: %v", err)
		// Не фатально — продолжаем
	}

	return c.Reply(response, &telebot.SendOptions{ParseMode: "Markdown"})
}

// HandleText обрабатывает текстовые сообщения.
// HandleText обрабатывает любые текстовые сообщения, не являющиеся командами.
func (b *BotService) HandleText(c telebot.Context) error {
	helpMessage := `
Доступные команды:

/start – регистрация пользователя
/list – список сохранённых встреч
/get [номер] – получить текст встречи
/find [запрос] – поиск встречи по ключевым словам
/chat [вопрос] – задать вопрос GigaChat

Также вы можете отправить голосовое сообщение или аудиофайл для транскрипции.
	`

	// return c.Reply(strings.TrimSpace(helpMessage), &telebot.SendOptions{
	return c.Send(strings.TrimSpace(helpMessage), &telebot.SendOptions{
		ParseMode: "Markdown",
	})
}

// HandleStart обрабатывает команду /start.
func (b *BotService) HandleStart(c telebot.Context) error {
	ctx := b.getCtx(c)
	user := c.Sender()

	existingUser, err := b.UserRepo.FindByTelegramID(ctx, user.ID)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Error checking user existence: %v", err)
	}

	userDB := &models.User{
		TelegramID: user.ID,
		Username:   &user.Username,
		FirstName:  &user.FirstName,
		LastName:   &user.LastName,
	}

	var message string

	if existingUser == nil {
		err = b.UserRepo.CreateIfNotExists(ctx, userDB)
		if err != nil {
			b.Logger.Error().Err(err).Msgf("Failed to save new user: %v", err)
		}
		message = fmt.Sprintf("Добро пожаловать, %s!\nТы успешно зарегистрирован.", user.FirstName)
	} else {
		userDB.ID = existingUser.ID
		err = b.UserRepo.Update(ctx, userDB)
		if err != nil {
			b.Logger.Error().Err(err).Msgf("Failed to update user: %v", err)
		}
		message = fmt.Sprintf("С возвращением, %s!\nРад снова тебя видеть.", user.FirstName)
	}

	return c.Reply(message, &telebot.SendOptions{
		ReplyMarkup: MenuInBot,
	})
}
