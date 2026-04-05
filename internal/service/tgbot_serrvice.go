package service

import (
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

// notifyUserOfFailure отправляет пользователю сообщение о сбое обработки
func (b *Bot) notifyUserOfFailure(userID int64, meetingTitle string) {
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

func (b *Bot) notifyUserOfSuccess(userID int64, title string) {
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
func (b *Bot) HandleStart(c telebot.Context) error {
	user := c.Sender()

	// Проверяем, существует ли пользователь
	existingUser, err := b.UserRepo.FindByTelegramID(user.ID)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		// Продолжаем, даже если ошибка - возможно, БД временно недоступна
	}

	// Создаём модель для сохранения/обновления
	userDB := &models.User{
		TelegramID: user.ID,
		Username:   &user.Username,
		FirstName:  &user.FirstName,
		LastName:   &user.LastName,
	}

	var message string

	if existingUser == nil {
		err = b.UserRepo.CreateIfNotExists(userDB) // Пользователя нет - регистрируем нового
		if err != nil {
			log.Printf("Failed to save new user: %v", err) // всё равно отправим приветствие
		}
		message = fmt.Sprintf("Добро пожаловать, %s!\nТы успешно зарегистрирован.", user.FirstName)
	} else {
		// Обновляем данные и время последнего визита
		userDB.ID = existingUser.ID // нужно для обновления
		err = b.UserRepo.Update(userDB)
		if err != nil {
			log.Printf("Failed to update user: %v", err)
		}
		message = fmt.Sprintf("С возвращением, %s!\nРад снова тебя видеть.", user.FirstName)
	}

	return c.Send(message)
}

// /get 1
func (b *Bot) HandleGet(c telebot.Context) error {
	user := c.Sender()
	args := c.Args()

	if len(args) == 0 {
		return c.Send("Укажите номер встречи. Пример: /get 1")
	}

	// Парсим номер
	index, err := strconv.Atoi(args[0])
	if err != nil {
		return c.Send("Неверный формат номера. Укажите число.")
	}

	if index < 1 {
		return c.Send("Номер должен быть больше 0.")
	}

	// Получаем все встречи
	meetings, err := b.MeetingRepo.ListByUser(user.ID)
	if err != nil {
		log.Printf("Failed to fetch meetings: %v", err)
		return c.Send("Не удалось загрузить список встреч.")
	}

	// Проверяем диапазон
	if index > len(meetings) {
		return c.Send(fmt.Sprintf("Нет встречи с номером %d. Доступно встреч: %d.", index, len(meetings)))
	}

	// Берём встречу по индексу
	meeting := meetings[index-1] // т.к. пользователь вводит 1-based

	// Теперь можно загрузить полные данные: транскрипцию и выжимку
	fullMeeting, err := b.MeetingRepo.GetByUserAndID(user.ID, meeting.ID)
	if err != nil {
		log.Printf("Failed to load full meeting %s: %v", meeting.ID, err)
		return c.Send("Не удалось загрузить содержимое встречи.")
	}

	if fullMeeting == nil {
		return c.Send("Встреча не найдена или доступ запрещён.")
	}

	// Формируем ответ
	var result strings.Builder
	result.WriteString(fmt.Sprintf("*%s*\n\n", fullMeeting.Title))

	if fullMeeting.SummaryText != nil {
		result.WriteString("*Краткая выжимка:*\n")
		result.WriteString(escapeMarkdown(*fullMeeting.SummaryText))
	} else {
		result.WriteString("*Краткая выжимка:* ещё не готова.\n\n")
	}

	if fullMeeting.TranscriptionText != nil {
		result.WriteString("*Транскрипция:*\n")
		result.WriteString(escapeMarkdown(*fullMeeting.TranscriptionText))
		result.WriteString("\n\n")
	} else {
		result.WriteString("*Транскрипция:* ещё не готова.\n\n")
	}

	return c.Send(result.String(), &telebot.SendOptions{ParseMode: "Markdown"})
}

// HandleFind обрабатывает команду /find "запрос"
func (b *Bot) HandleFind(c telebot.Context) error {
	user := c.Sender()
	args := c.Args()

	if len(args) == 0 {
		return c.Send("Введите запрос для поиска.\nПример: /find встреча с клиентом")
	}

	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return c.Send("Запрос не может быть пустым.")
	}

	// Логируем запрос
	log.Printf("User %d searching for: %q", user.ID, query)

	// 1. Выполняем поиск
	meetings, err := b.MeetingRepo.SearchByUser(user.ID, query)
	if err != nil {
		log.Printf("Search failed for user %d: %v", user.ID, err)
		return c.Send("Произошла ошибка при поиске.")
	}

	// 2. Если ничего не найдено
	if len(meetings) == 0 {
		return c.Send(fmt.Sprintf("Ничего не найдено по запросу:\n`%s`", escapeMarkdown(query)),
			&telebot.SendOptions{ParseMode: "Markdown"})
	}

	// 3. Формируем список результатов
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

	// 4. Отправляем результат
	message := fmt.Sprintf(
		"Найдено %d результатов по запросу _%s_:\n\n%s\n\n"+
			"Чтобы открыть используйте /get [номер]",
		len(meetings),
		escapeMarkdown(query),
		strings.Join(items, "\n\n"),
	)

	return c.Send(message, &telebot.SendOptions{
		ParseMode: "Markdown",
	})
}

// HandleList обрабатывает команду /list.
func (b *Bot) HandleList(c telebot.Context) error {
	user := c.Sender()

	// 1. Получаем встречи пользователя
	meetings, err := b.MeetingRepo.ListByUser(user.ID)
	if err != nil {
		log.Printf("Failed to fetch meetings for user %d: %v", user.ID, err)
		return c.Send("Не удалось загрузить список встреч.")
	}

	// 2. Если нет встреч
	if len(meetings) == 0 {
		return c.Send("У вас пока нет сохранённых встреч.\nОтправьте голосовое сообщение, и оно появится здесь.")
	}

	// 3. Формируем список
	var items []string
	for i, m := range meetings {
		// Форматируем дату: 05.04.2025 14:30
		dateStr := m.CreatedAt.Format("02.01.2006 15:04")
		// Статус: uploaded - Загружено, completed - Готово
		statusText := formatStatus(m.Status)
		// Добавляем элемент
		items = append(items, fmt.Sprintf(
			"%d. *%s*\n   Дата: %s | Статус: %s",
			i+1,
			escapeMarkdown(m.Title),
			dateStr,
			statusText,
		))
	}

	// 4. Собираем сообщение
	message := fmt.Sprintf(
		"Ваши встречи (%d):\n\n%s\n\n"+
			"Чтобы получить текст встречи, используйте команду /get [номер]",
		len(meetings),
		strings.Join(items, "\n\n"),
	)

	// 5. Отправляем с Markdown
	return c.Send(message, &telebot.SendOptions{
		ParseMode: "Markdown",
	})
}
