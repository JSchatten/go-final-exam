// internal/worker/transcription_worker.go
package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/integration/gigachat"
	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

// TranscriptionWorker проверяет статус задач распознавания и при завершении
// обновляет транскрипцию и сохраняет краткую выжимку.
type TranscriptionWorker struct {
	transcriptionRepo *repository.TranscriptionRepository
	summaryRepo       *repository.SummaryRepository
	saluteSpeech      *salutespeech.SaluteSpeechClient
	gigaChat          *gigachat.GigaChatClient
	ticker            *time.Ticker
}

// NewTranscriptionWorker создаёт новый воркер.
func NewTranscriptionWorker(
	transcriptionRepo *repository.TranscriptionRepository,
	summaryRepo *repository.SummaryRepository,
	saluteSpeech *salutespeech.SaluteSpeechClient,
	gigaChat *gigachat.GigaChatClient,
) *TranscriptionWorker {
	return &TranscriptionWorker{
		transcriptionRepo: transcriptionRepo,
		summaryRepo:       summaryRepo,
		saluteSpeech:      saluteSpeech,
		gigaChat:          gigaChat,
		ticker:            time.NewTicker(5 * time.Second),
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
	transcriptions, err := w.transcriptionRepo.GetUnprocessed(ctx)
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
	taskResult, err := w.saluteSpeech.CheckTaskStatus(t.SaluteTaskID)
	if err != nil {
		return err
	}

	newStatus := taskResult.Status
	var fullText *string

	switch newStatus {
	case "DONE":
		// Получаем результат распознавания
		recognitionResults, err := w.saluteSpeech.GetRecognitionResult(taskResult.ResponseFileID)
		if err != nil {
			return err
		}

		text := recognitionResults.GetFullNormalizedText()
		fullText = &text

		// Обновляем транскрипцию
		err = w.transcriptionRepo.Update(ctx, t.ID, newStatus, fullText)
		if err != nil {
			return fmt.Errorf("failed to update transcription: %w", err)
		}

		// Генерируем краткую выжимку
		summaryText, err := w.gigaChat.Transcribe(text)
		if err != nil {
			log.Printf("TranscriptionWorker: failed to generate summary for meeting %s: %v", t.MeetingID, err)
			return nil // Не прерываем — ошибка генерации не фатальна
		}

		// Сохраняем выжимку
		_, err = w.summaryRepo.Create(t.MeetingID, summaryText)
		if err != nil {
			log.Printf("TranscriptionWorker: failed to save summary: %v", err)
			return nil // Ошибка сохранения — логируем, но не падаем
		}

		log.Printf("TranscriptionWorker: generated and saved summary for meeting %s", t.MeetingID)

	case "ERROR", "CANCELED":
		log.Printf("TranscriptionWorker: task %s failed with status %s", t.SaluteTaskID, newStatus)

		// Обновляем только транскрипцию
		err = w.transcriptionRepo.Update(ctx, t.ID, newStatus, nil)
		if err != nil {
			return fmt.Errorf("failed to update transcription: %w", err)
		}
	}

	return nil
}
