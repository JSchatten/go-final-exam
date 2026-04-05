package service

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"gopkg.in/telebot.v3"
)

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
