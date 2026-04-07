// internal/worker/transcription_worker.go
package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/JSchatten/go-final-exam/internal/service"
)

// TranscriptionWorker проверяет статус задач распознавания и при завершении
// обновляет транскрипцию и сохраняет краткую выжимку.
type TranscriptionWorker struct {
	bot    *service.BotService
	ticker *time.Ticker
}

// NewTranscriptionWorker создаёт новый воркер.
func NewTranscriptionWorker(
	bot *service.BotService,
) *TranscriptionWorker {
	return &TranscriptionWorker{
		bot:    bot,
		ticker: time.NewTicker(5 * time.Second),
	}
}

// Start запускает фоновую проверку статусов.
func (w *TranscriptionWorker) Start(ctx context.Context) error {
	log.Println("TranscriptionWorker: started checking transcription statuses every 5 seconds")
	defer w.ticker.Stop()

	for {
		select {
		case <-w.ticker.C:
			if err := w.pollAndSync(ctx); err != nil {
				log.Printf("TranscriptionWorker: failed to sync statuses: %v", err)
			}
		case <-ctx.Done():
			log.Println("TranscriptionWorker: shutting down...")
			return ctx.Err()
		}
	}
}

// pollAndSync получает необработанные транскрипции и синхронизирует их статус.
func (w *TranscriptionWorker) pollAndSync(ctx context.Context) error {
	transcriptions, err := w.bot.TranscriptionRepo.GetUnprocessed(ctx)
	if err != nil {
		return err
	}

	log.Printf("TranscriptionWorker: found %d unprocessed transcription(s)", len(transcriptions))

	for _, t := range transcriptions {
		if err := w.syncTranscription(ctx, t); err != nil {
			log.Printf("TranscriptionWorker: failed to sync transcription %s: %v", t.ID, err)
		}
	}

	return nil
}

// syncTranscription проверяет статус одной транскрипции и обновляет данные.
func (w *TranscriptionWorker) syncTranscription(ctx context.Context, t *models.Transcription) error {
	taskResult, err := w.bot.SaluteSpeech.CheckTaskStatus(t.SaluteTaskID)
	if err != nil {
		return err
	}

	newStatus := taskResult.Status
	var fullText *string

	switch newStatus {
	case "DONE":
		// Получаем результат распознавания
		recognitionResults, err := w.bot.SaluteSpeech.GetRecognitionResult(taskResult.ResponseFileID)
		if err != nil {
			return err
		}

		text := recognitionResults.GetFullNormalizedText()
		fullText = &text

		// Обновляем транскрипцию
		err = w.bot.TranscriptionRepo.Update(ctx, t.ID, newStatus, fullText)
		if err != nil {
			return fmt.Errorf("failed to update transcription: %w", err)
		}

		// Генерируем краткую выжимку
		summaryText, err := w.bot.GigaChat.Transcribe(text)
		if err != nil {
			log.Printf("TranscriptionWorker: failed to generate summary for meeting %s: %v", t.MeetingID, err)
			// Прерываем, ошибка генерации фатальна для кошелька на повторные запросы
			return err
		} else {
			// Сохраняем выжимку
			_, err = w.bot.SummaryRepo.Create(t.MeetingID, summaryText)
			if err != nil {
				log.Printf("TranscriptionWorker: failed to save summary: %v", err)
				// Ошибка сохранения, что-то не то с БД
				return err
			}
		}

		err = w.bot.MeetingRepo.UpdateStatusWithError(ctx, t.MeetingID, models.MeetingStatusCompleted, "")
		if err != nil {
			log.Printf("TranscriptionWorker: failed to update meeting status to 'failed': %v", err)
		}

		log.Printf("TranscriptionWorker: generated and saved summary for meeting %s", t.MeetingID)

	case "ERROR", "CANCELED":
		log.Printf("TranscriptionWorker: task %s failed with status %s", t.SaluteTaskID, newStatus)
		// Обновляем только транскрипцию
		err = w.bot.TranscriptionRepo.Update(ctx, t.ID, newStatus, nil)
		if err != nil {
			return fmt.Errorf("failed to update transcription: %w", err)
		}
		// Обновляем статус встречи на "failed"
		err = w.bot.MeetingRepo.UpdateStatusWithError(ctx, t.MeetingID, models.MeetingStatusFailed, models.ErrorTranscriptionFailed)
		if err != nil {
			log.Printf("TranscriptionWorker: failed to update meeting status to 'failed': %v", err)
		}
	}

	return nil
}
