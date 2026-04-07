package service

import (
	"strings"

	"github.com/JSchatten/go-final-exam/internal/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Вспомогательная функция: читаемое название статуса
// TODO: надо выделить рядом с константами
func formatStatus(status string) string {
	switch status {
	case models.MeetingStatusUploaded:
		return "Загружено"
	case models.MeetingStatusProcessing:
		return "Обрабатывается"
	case models.MeetingStatusCompleted:
		return "Готово"
	case models.MeetingStatusFailed:
		return "Ошибка"
	default:
		return cases.Title(language.Und).String(status)
	}
}

// Экранируем специальные символы для Markdown
func escapeMarkdown(text string) string {
	for _, ch := range []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"} {
		text = strings.ReplaceAll(text, ch, "\\"+ch)
	}
	return text
}
