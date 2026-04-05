// internal/service/list_service.go
package service

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/telebot.v3"
)

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

		// Статус: uploaded → Загружено, completed → Готово
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

// Вспомогательная функция: читаемое название статуса
// TODO: надо выделить рядом с константами
func formatStatus(status string) string {
	switch status {
	case "uploaded":
		return "Загружено"
	case "processing":
		return "Обрабатывается"
	case "completed":
		return "Готово"
	case "failed":
		return "Ошибка"
	default:
		return strings.Title(status)
	}
}

// Экранируем специальные символы для Markdown
func escapeMarkdown(text string) string {
	for _, ch := range []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"} {
		text = strings.ReplaceAll(text, ch, "\\"+ch)
	}
	return text
}
