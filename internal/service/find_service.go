// internal/service/find_service.go
package service

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/telebot.v3"
)

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
