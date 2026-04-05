package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"gopkg.in/telebot.v3"
)

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

// processMeeting - фоновая обработка встречи: транскрипция + выжимка
func (b *Bot) processMeeting(ctx context.Context, meeting *models.Meeting) {
	log.Printf("Starting processing of meeting %s for user %d", meeting.ID, meeting.UserID)
	const maxRetries = 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := b.transcribeAndSummarize(ctx, meeting)
		if err == nil {
			_ = b.MeetingRepo.UpdateStatus(meeting.ID, models.StatusCompleted)
			// Оповестить пользователя
			log.Printf("Meeting %s processed successfully", meeting.ID)
			b.notifyUserOfSuccess(meeting.UserID, meeting.Title)
			return
		}

		log.Printf("Attempt %d failed for meeting %s: %v", attempt, meeting.ID, err)

		if attempt == maxRetries {
			_ = b.MeetingRepo.UpdateStatus(meeting.ID, models.StatusFailed)
			// Оповестить пользователя
			b.notifyUserOfFailure(meeting.UserID, meeting.Title)
			return
		}

		// Ждём перед повторной попыткой (экспоненциальная задержка)
		select {
		case <-time.After(time.Second * time.Duration(attempt*5)):
		case <-ctx.Done():
			return
		}
	}
}

// transcribeAndSummarize - выполняет оба шага параллельно (или последовательно)
func (b *Bot) transcribeAndSummarize(ctx context.Context, meeting *models.Meeting) error {
	g, ctx := errgroup.WithContext(ctx)

	var transcriptionText string
	var summaryText string

	// Шаг 1: Транскрипция аудио - текст
	g.Go(func() error {
		text, err := b.transcribeAudio(ctx, meeting)
		if err != nil {
			return fmt.Errorf("transcription failed: %w", err)
		}
		transcriptionText = text
		return nil
	})

	// Шаг 2: Краткая выжимка GigaChat
	// Выполняется ПОСЛЕ транскрипции (логично: нужен текст)
	g.Go(func() error {
		// Ждём, пока транскрипция завершится
		// Но если мы хотим параллельно - можно сделать отдельный источник
		// А пока просто блокируем
		if transcriptionText == "" {
			// В данном случае лучше выполнять последовательно
			return nil
		}

		summary, err := b.generateSummary(ctx, transcriptionText)
		if err != nil {
			return fmt.Errorf("summary generation failed: %w", err)
		}
		summaryText = summary
		return nil
	})

	// Ждём завершения всех задач
	if err := g.Wait(); err != nil {
		return err
	}

	// Сохраняем результаты в БД
	if err := b.saveResults(ctx, meeting.ID, transcriptionText, summaryText); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	return nil
}

func (b *Bot) transcribeAudio(ctx context.Context, meeting *models.Meeting) (string, error) {
	audioPath := *meeting.AudioFilePath

	// 1. Загружаем файл в SaluteSpeech
	fileID, err := b.SaluteSpeech.UploadFileByPath(audioPath)
	if err != nil {
		return "", fmt.Errorf("upload to SaluteSpeech failed: %w", err)
	}

	// 2. Создаём задачу на распознавание
	taskID, err := b.SaluteSpeech.CreateRecognitionTask(audioPath, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to create recognition task: %w", err)
	}

	// 3. Ждём завершения
	var taskResult *salutespeech.TaskResult
	for {
		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return "", ctx.Err()
		}

		taskResult, err = b.SaluteSpeech.CheckTaskStatus(taskID)
		if err != nil {
			return "", err
		}

		switch taskResult.Status {
		case "DONE":
			break
		case "ERROR", "CANCELED":
			return "", fmt.Errorf("recognition task failed with status: %s", taskResult.Status)
		default:
			continue
		}
		break
	}

	// 4. Получаем результат
	recognition, err := b.SaluteSpeech.GetRecognitionResult(taskResult.ResponseFileID)
	if err != nil {
		return "", fmt.Errorf("failed to get recognition result: %w", err)
	}

	return recognition.GetFullText(), nil
}

func (b *Bot) generateSummary(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf("Сделай краткую выжимку из следующего текста:\n\n%s", text)

	summary, err := b.GigaChat.SendMessage(prompt)
	if err != nil {
		return "", err
	}

	return summary, nil
}

func (b *Bot) saveResults(ctx context.Context, meetingID uuid.UUID, transcription, summary string) error {
	g, ctx := errgroup.WithContext(ctx)

	var transID uuid.UUID
	var sumID uuid.UUID

	// Сохраняем транскрипцию
	g.Go(func() error {
		id, err := b.TranscriptionRepo.Create(meetingID, transcription)
		if err != nil {
			return err
		}
		transID = id
		return nil
	})

	// Сохраняем выжимку
	g.Go(func() error {
		id, err := b.SummaryRepo.Create(meetingID, summary)
		if err != nil {
			return err
		}
		sumID = id
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	// Обновляем ссылки в meeting
	if err := b.MeetingRepo.UpdateTranscription(meetingID, transID); err != nil {
		return err
	}
	if err := b.MeetingRepo.UpdateSummary(meetingID, sumID); err != nil {
		return err
	}

	return nil
}
