package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/telebot.v3"
)

// HandleGet обрабатывает команду /get [номер]
func (b *BotService) HandleGet(c telebot.Context) error {
	b.Logger.Debug().Msg("HandleGet start")
	ctx := b.getCtx(c)
	user := c.Sender()

	// Получаем аргументы: /get 123
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

	// Получаем все встречи пользователя
	meetings, err := b.MeetingRepo.ListByUser(ctx, user.ID)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to fetch meetings: %v", err)
		return c.Reply("Не удалось загрузить список встреч.")
	}

	if index > len(meetings) {
		return c.Reply(fmt.Sprintf("Нет встречи с номером %d. Доступно встреч: %d.", index, len(meetings)))
	}

	// Берём встречу по индексу
	meeting := meetings[index-1]

	// Загружаем полные данные по ID
	fullMeeting, err := b.MeetingRepo.GetByUserAndID(ctx, user.ID, meeting.ID)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to load full meeting %s: %v", meeting.ID, err)
		return c.Reply("Не удалось загрузить содержимое встречи.")
	}

	if fullMeeting == nil {
		return c.Reply("Встреча не найдена или доступ запрещён.")
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("*%s*\n\n", fullMeeting.Title))

	if fullMeeting.SummaryText != nil {
		result.WriteString("*Краткая выжимка:*\n")
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

func (b *BotService) HandleGetByID(c telebot.Context) error {
	ctx := b.getCtx(c)
	user := c.Sender()

	// Получаем meeting_id из контекста
	meetingID, ok := c.Get("meeting_id").(uuid.UUID)
	if !ok {
		return c.Reply("Не указан идентификатор встречи.")
	}

	fullMeeting, err := b.MeetingRepo.GetByUserAndID(ctx, user.ID, meetingID)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to load meeting by ID: %s", meetingID)
		return c.Reply("Не удалось загрузить содержимое встречи.")
	}
	if fullMeeting == nil {
		return c.Reply("Встреча не найдена или доступ запрещён.")
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("*%s*\n\n", fullMeeting.Title))

	if fullMeeting.SummaryText != nil {
		result.WriteString("*Краткая выжимка:*\n")
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

	b.Logger.Info().Msgf("User %d searching for: %q", user.ID, query)

	meetings, err := b.MeetingRepo.SearchByUser(ctx, user.ID, query)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Search failed for user %d: %v", user.ID, err)
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

// HandleList обрабатывает команду /list
func (b *BotService) HandleList(c telebot.Context) error {
	b.Logger.Debug().Msgf("HandleList start")
	ctx := b.getCtx(c)
	user := c.Sender()

	// Получаем все встречи пользователя
	meetings, err := b.MeetingRepo.ListByUser(ctx, user.ID)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to fetch meetings for user %d", user.ID)
		return c.Reply("Не удалось загрузить список встреч.")
	}

	if len(meetings) == 0 {
		return c.Reply("У вас пока нет сохранённых встреч.\nОтправьте голосовое сообщение, и оно появится здесь.")
	}

	// Определяем текущую страницу
	page := 0
	callback := c.Callback()
	if callback != nil && callback.Data != "" {
		if strings.HasPrefix(callback.Data, "page:") {
			fmt.Sscanf(callback.Data, "page:%d", &page)
		}
	} else {
		b.Logger.Info().Msgf("callback == nil OR callback.Data = '' ")
	}

	totalPages := (len(meetings) + ItemsPerPage - 1) / ItemsPerPage
	if page < 0 || page >= totalPages {
		page = 0
	}

	from := page * ItemsPerPage
	to := min(from+ItemsPerPage, len(meetings))
	currentPage := meetings[from:to]

	var items []string
	for i, m := range currentPage {
		idx := from + i + 1
		dateStr := m.CreatedAt.Format("02.01 15:04")
		statusText := formatStatus(m.Status)
		items = append(items, fmt.Sprintf(
			"%d. *%s*\n   %s | %s",
			idx, escapeMarkdown(m.Title), dateStr, statusText,
		))
	}

	message := fmt.Sprintf(
		"Ваши встречи (%d):\n\n%s\n\n"+
			"Выберите номер встречи или переключайтесь между страницами.",
		len(meetings),
		strings.Join(items, "\n\n"),
	)

	// Создаём НОВЫЙ экземпляр ReplyMarkup каждый раз
	paginator := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	// Первая строка: кнопки 1–5 (отображаем номер, но в Data - ID встречи)
	var numBtns []telebot.Btn
	for i, m := range currentPage {
		btn := paginator.Data(
			fmt.Sprintf("%d", from+i+1),          // отображается: "1", "2", ...
			fmt.Sprintf("get:%s", m.ID.String()), // передаётся: "get:uuid"
		)
		numBtns = append(numBtns, btn)
	}
	if len(numBtns) > 0 {
		rows = append(rows, paginator.Row(numBtns...))
	}

	// Вторая строка: навигация
	var navBtns []telebot.Btn
	if page > 0 {
		navBtns = append(navBtns, paginator.Data("Назад", fmt.Sprintf("page:%d", page-1)))
	}
	if page < totalPages-1 {
		navBtns = append(navBtns, paginator.Data("Вперёд", fmt.Sprintf("page:%d", page+1)))
	}
	navBtns = append(navBtns, paginator.Data("В начало", "/start"))
	rows = append(rows, paginator.Row(navBtns...))

	// Применяем разметку
	paginator.Inline(rows...)

	// Если это колбэк - редактируем сообщение
	if callback != nil {
		b.Logger.Debug().Msg("HandleList callback captured")
		currentMsg := c.Message()

		// Проверяем, отличается ли новое сообщение от текущего
		if currentMsg.Text == message && isSameInlineMarkup(currentMsg.ReplyMarkup, paginator) {
			// Ничего не изменилось - просто подтверждаем нажатие
			b.Logger.Debug().Msg("HandleList callback nothing changed")
			return c.Respond()
		}

		// Пробуем отредактировать
		b.Logger.Debug().Msg("HandleList callback try edit")
		err := c.Edit(message, &telebot.SendOptions{
			ParseMode:   "Markdown",
			ReplyMarkup: paginator,
		})
		if err != nil {
			if strings.Contains(err.Error(), "message is not modified") {
				return c.Respond()
			}
			return err
		}
		return nil
	} else {
		// Первый вызов - отправляем новое сообщение
		b.Logger.Debug().Msg("HandleList callback first cal")
		return c.Reply(message, &telebot.SendOptions{
			ParseMode:   "Markdown",
			ReplyMarkup: paginator,
		})
	}
}

func isSameInlineMarkup(a, b *telebot.ReplyMarkup) bool {

	aRow := a.InlineKeyboard
	bRow := b.InlineKeyboard

	if a == nil || b == nil || len(aRow) != len(bRow) {
		return false
	}
	for i := range aRow {
		rowA := aRow[i]
		rowB := bRow[i]
		if len(rowA) != len(rowB) {
			return false
		}
		for j := range rowA {
			if rowA[j].Data != rowB[j].Data {
				return false
			}
		}
	}
	return true
}
