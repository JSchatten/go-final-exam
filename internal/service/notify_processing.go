package service

import (
	"fmt"

	"gopkg.in/telebot.v3"
)

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
		b.Logger.Error().Err(err).Msgf("Failed to send failure notification to user %d: %v", userID, err)
		return
	}

	// Опционально: можно залогировать ID отправленного сообщения
	b.Logger.Warn().Msgf("Failure notification sent to user %d, message ID: %d", userID, msg.ID)
}

func (b *BotService) notifyUserOfSuccess(userID int64, title string) {
	msg := fmt.Sprintf("Встреча обработана:\n\n*%s*\n\nТеперь доступна в /list", escapeMarkdown(title))
	recipient := &telebot.User{ID: userID}

	message, err := b.Telebot.Send(recipient, msg, &telebot.SendOptions{
		ParseMode: "Markdown",
	})
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to send success notification to user %d: %v", userID, err)
		return
	}
	b.Logger.Info().Msgf("Success notification sent to user %d, message ID: %d", userID, message.ID)
}
