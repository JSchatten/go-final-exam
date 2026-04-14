package service

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/telebot.v3"
)

// /get 1
func (b *BotService) HandleGet(c telebot.Context) error {
	b.Logger.Debug().Msg("HandleGet start")
	ctx := b.getCtx(c)
	user := c.Sender()

	// Пытаемся получить args из c.Get("args"), иначе используем c.Args()
	var args []string
	if savedArgs, ok := c.Get("args").([]string); ok {
		args = savedArgs
	} else {
		args = c.Args()
	}
	b.Logger.Debug().Msgf("args: %v", args)

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
		b.Logger.Error().Err(err).Msgf("Failed to fetch meetings: %v", err)
		return c.Reply("Не удалось загрузить список встреч.")
	}

	if index > len(meetings) {
		return c.Reply(fmt.Sprintf("Нет встречи с номером %d. Доступно встреч: %d.", index, len(meetings)))
	}

	meeting := meetings[index-1]

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
	ctx := b.getCtx(c)
	user := c.Sender()

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

	// Создаём разметку
	paginator := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	// Первая строка: кнопки 1–5 (номера встреч на этой странице)
	var numBtns []telebot.Btn
	for i := 0; i < ItemsPerPage; i++ {
		idx := from + i + 1
		if idx > len(meetings) {
			break
		}
		btn := paginator.Data(fmt.Sprintf("%d", idx), fmt.Sprintf("get:%d", idx))
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

	paginator.Inline(rows...)

	// Отправляем сообщение
	if callback != nil {
		return c.Edit(message, &telebot.SendOptions{
			ParseMode:   "Markdown",
			ReplyMarkup: paginator,
		})
	} else {
		return c.Reply(message, &telebot.SendOptions{
			ParseMode:   "Markdown",
			ReplyMarkup: paginator,
		})
	}
}
