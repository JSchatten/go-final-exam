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
	case models.StatusUploaded:
		return "Загружено"
	case models.StatusProcessing:
		return "Обрабатывается"
	case models.StatusCompleted:
		return "Готово"
	case models.StatusFailed:
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
