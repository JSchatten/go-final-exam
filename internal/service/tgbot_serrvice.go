package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gopkg.in/telebot.v3"

	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/JSchatten/go-final-exam/internal/repository"
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
}

// NewBotService создаёт новый экземпляр бота с готовыми зависимостями
func NewBotService(
	bot *telebot.Bot,
	gigaChat *gigachat.GigaChatClient,
	saluteSpeech *salutespeech.SaluteSpeechClient,
	db *repository.DB,
	audioStoragePath string,
) *BotService {

	MenuInBot.Reply(
		MenuInBot.Row(btnHelp),
		MenuInBot.Row(BtnStart),
		MenuInBot.Row(btnFind),
	)

	return &BotService{
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
	}
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

	response, err := b.GigaChat.SendMessage(prompt)
	if err != nil {
		log.Printf("Failed to get response from GigaChat: %v", err)
		return c.Reply("Не удалось получить ответ от GigaChat.")
	}

	// Сохраняем в историю
	ctx := b.getCtx(c)
	err = b.ChatHistoryRepo.Create(ctx, c.Sender().ID, prompt, response)
	if err != nil {
		log.Printf("Failed to save chat history: %v", err)
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

// notifyUserOfFailure отправляет пользователю сообщение о сбое обработки
func (b *BotService) notifyUserOfFailure(userID int64, meetingTitle string) {
	messageText := fmt.Sprintf(
		"Не удалось обработать встречу:\n\n"+
			"*%s*\n\n"+
			"К сожалению, произошла ошибка при распознавании или создании выжимки.\n"+
			"Попробуйте отправить аудиозапись ещё раз.",
		escapeMarkdown(meetingTitle),
	)

	// Создаём объект пользователя для отправки
	recipient := &telebot.User{
		ID: userID,
	}

	// Отправляем сообщение
	msg, err := b.Telebot.Send(recipient, messageText, &telebot.SendOptions{
		ParseMode: "Markdown",
	})

	if err != nil {
		log.Printf("Failed to send failure notification to user %d: %v", userID, err)
		return
	}

	// Опционально: можно залогировать ID отправленного сообщения
	log.Printf("Failure notification sent to user %d, message ID: %d", userID, msg.ID)
}

func (b *BotService) notifyUserOfSuccess(userID int64, title string) {
	msg := fmt.Sprintf("Встреча обработана:\n\n*%s*\n\nТеперь доступна в /list", escapeMarkdown(title))
	recipient := &telebot.User{ID: userID}

	message, err := b.Telebot.Send(recipient, msg, &telebot.SendOptions{
		ParseMode: "Markdown",
	})
	if err != nil {
		log.Printf("Failed to send success notification to user %d: %v", userID, err)
		return
	}
	log.Printf("Success notification sent to user %d, message ID: %d", userID, message.ID)
}

// HandleStart обрабатывает команду /start.
func (b *BotService) HandleStart(c telebot.Context) error {
	ctx := b.getCtx(c)
	user := c.Sender()

	existingUser, err := b.UserRepo.FindByTelegramID(ctx, user.ID)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
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
			log.Printf("Failed to save new user: %v", err)
		}
		message = fmt.Sprintf("Добро пожаловать, %s!\nТы успешно зарегистрирован.", user.FirstName)
	} else {
		userDB.ID = existingUser.ID
		err = b.UserRepo.Update(ctx, userDB)
		if err != nil {
			log.Printf("Failed to update user: %v", err)
		}
		message = fmt.Sprintf("С возвращением, %s!\nРад снова тебя видеть.", user.FirstName)
	}

	return c.Reply(message, &telebot.SendOptions{
		ReplyMarkup: MenuInBot,
	})
}

// /get 1
func (b *BotService) HandleGet(c telebot.Context) error {
	ctx := b.getCtx(c)
	user := c.Sender()
	args := c.Args()

	if len(args) == 0 {
		return c.Reply("Укажите номер встречи. Пример: /get 1")
	}

	index, err := strconv.Atoi(args[0])
	if err != nil {
		return c.Reply("Неверный формат номера. Укажите число.")
	}

	if index < 1 {
		return c.Reply("Номер должен быть больше 0.")
	}

	meetings, err := b.MeetingRepo.ListByUser(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to fetch meetings: %v", err)
		return c.Reply("Не удалось загрузить список встреч.")
	}

	if index > len(meetings) {
		return c.Reply(fmt.Sprintf("Нет встречи с номером %d. Доступно встреч: %d.", index, len(meetings)))
	}

	meeting := meetings[index-1]

	fullMeeting, err := b.MeetingRepo.GetByUserAndID(ctx, user.ID, meeting.ID)
	if err != nil {
		log.Printf("Failed to load full meeting %s: %v", meeting.ID, err)
		return c.Reply("Не удалось загрузить содержимое встречи.")
	}

	if fullMeeting == nil {
		return c.Reply("Встреча не найдена или доступ запрещён.")
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("*%s*\n\n", fullMeeting.Title))

	if fullMeeting.SummaryText != nil {
		result.WriteString("*Краткая выжимка:*\n")
		// result.WriteString(escapeMarkdown(*fullMeeting.SummaryText))
		result.WriteString(*fullMeeting.SummaryText)
	} else {
		result.WriteString("*Краткая выжимка:* ещё не готова.\n\n")
	}

	if fullMeeting.TranscriptionText != nil {
		result.WriteString("\n*Транскрипция:*\n")
		result.WriteString(*fullMeeting.TranscriptionText)
		result.WriteString("\n")
	} else {
		result.WriteString("*Транскрипция:* ещё не готова.\n\n")
	}

	return c.Reply(result.String(), &telebot.SendOptions{ParseMode: "Markdown"})
}

// HandleFind обрабатывает команду /find "запрос"
func (b *BotService) HandleFind(c telebot.Context) error {
	ctx := b.getCtx(c)
	user := c.Sender()
	args := c.Args()

	if len(args) == 0 {
		return c.Reply("Введите запрос для поиска.\nПример: /find встреча с клиентом")
	}

	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return c.Reply("Запрос не может быть пустым.")
	}

	log.Printf("User %d searching for: %q", user.ID, query)

	meetings, err := b.MeetingRepo.SearchByUser(ctx, user.ID, query)
	if err != nil {
		log.Printf("Search failed for user %d: %v", user.ID, err)
		return c.Reply("Произошла ошибка при поиске.")
	}

	if len(meetings) == 0 {
		return c.Reply(fmt.Sprintf("Ничего не найдено по запросу:\n`%s`", escapeMarkdown(query)),
			&telebot.SendOptions{ParseMode: "Markdown"})
	}

	var items []string
	for i, m := range meetings {
		dateStr := m.CreatedAt.Format("02.01.2006 15:04")
		statusText := formatStatus(m.Status)
		items = append(items, fmt.Sprintf(
			"%d. *%s*\n   Дата: %s | Статус: %s",
			i+1,
			escapeMarkdown(m.Title),
			dateStr,
			statusText,
		))
	}

	message := fmt.Sprintf(
		"Найдено %d результатов по запросу _%s_:\n\n%s\n\n"+
			"Чтобы открыть используйте /get [номер]",
		len(meetings),
		escapeMarkdown(query),
		strings.Join(items, "\n\n"),
	)

	return c.Reply(message, &telebot.SendOptions{ParseMode: "Markdown"})
}

// HandleList обрабатывает команду /list.
func (b *BotService) HandleList(c telebot.Context) error {
	ctx := b.getCtx(c)
	user := c.Sender()
	data := c.Callback()
	fmt.Printf("data %+v", data)
	if data != nil {
		fmt.Printf("data %+v", data.Data)
	}

	meetings, err := b.MeetingRepo.ListByUser(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to fetch meetings for user %d: %v", user.ID, err)
		return c.Reply("Не удалось загрузить список встреч.")
	}

	if len(meetings) == 0 {
		return c.Reply("У вас пока нет сохранённых встреч.\nОтправьте голосовое сообщение, и оно появится здесь.")
	}

	var items []string
	for i, m := range meetings {
		dateStr := m.CreatedAt.Format("02.01.2006 15:04")
		statusText := formatStatus(m.Status)
		items = append(items, fmt.Sprintf(
			"%d. *%s*\n   Дата: %s | Статус: %s",
			i+1,
			m.Title,
			dateStr,
			statusText,
		))
	}

	message := fmt.Sprintf(
		"Ваши встречи (%d):\n\n%s\n\n"+
			"Чтобы получить текст встречи, используйте команду /get [номер]",
		len(meetings),
		strings.Join(items, "\n\n"),
	)

	Paginator := &telebot.ReplyMarkup{}
	btnPrev := Paginator.Data("⬅", "prev") // Create a callback data button
	btnNext := Paginator.Data("➡", "next")
	btnOne := Selector.Data("1", "first", "")
	btnTwo := Selector.Data("2", "second", "")
	btnThree := Selector.Data("3", "third", "")
	btnFour := Selector.Data("4", "fourth", "")
	btnFive := Selector.Data("5", "fived", "")

	// Handler for the button
	b.Telebot.Handle("\ffirst", func(c telebot.Context) error {
		// Access the callback data (in this case "12345")
		data := c.Callback().Data

		// Respond to the callback
		// return c.Respond(&telebot.CallbackResponse{
		// 	Text: "Product " + data + " selected",
		// })

		Message := "Product " + data + " selected"

		return c.Reply(Message, &telebot.SendOptions{ParseMode: "Markdown"})
	})

	Paginator.Inline(
		Paginator.Row(btnPrev, btnNext),
		Paginator.Row(btnOne, btnTwo, btnThree, btnFour, btnFive),
	)

	return c.Reply(message, &telebot.SendOptions{
		ParseMode:   "Markdown",
		ReplyMarkup: Paginator,
	})
}
