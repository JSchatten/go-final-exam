package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

// TranscriptionWorker проверяет статус задач распознавания в SaluteSpeech и обновляет БД.
type TranscriptionWorker struct {
	repo         *repository.TranscriptionRepository
	saluteSpeech *salutespeech.SaluteSpeechClient
	ticker       *time.Ticker
}

// NewTranscriptionWorker создаёт новый воркер.
func NewTranscriptionWorker(repo *repository.TranscriptionRepository, saluteSpeech *salutespeech.SaluteSpeechClient) *TranscriptionWorker {
	return &TranscriptionWorker{
		repo:         repo,
		saluteSpeech: saluteSpeech,
		ticker:       time.NewTicker(5 * time.Second),
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

// pollAndSync получает необработанные транскрипции и синхронизирует их статус с SaluteSpeech
func (w *TranscriptionWorker) pollAndSync(ctx context.Context) error {
	transcriptions, err := w.repo.GetUnprocessed(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed transcriptions: %w", err)
	}

	log.Printf("TranscriptionWorker: found %d unprocessed transcription(s)", len(transcriptions))

	for _, t := range transcriptions {
		if err := w.syncTranscription(ctx, t); err != nil {
			log.Printf("TranscriptionWorker: failed to sync transcription %s: %v", t.ID, err)
			// Не прерываем цикл — продолжаем обработку других
		}
	}

	return nil
}

// syncTranscription проверяет статус одной транскрипции и обновляет её в БД
func (w *TranscriptionWorker) syncTranscription(ctx context.Context, t *models.Transcription) error {
	log.Printf("TranscriptionWorker: syncing task %s (status: %s)", t.SaluteTaskID, t.Status)

	taskResult, err := w.saluteSpeech.CheckTaskStatus(t.SaluteTaskID)
	if err != nil {
		return fmt.Errorf("failed to check task status: %w", err)
	}

	// Обновляем статус в БД
	newStatus := taskResult.Status // NEW, RUNNING, CANCELED, DONE, ERROR
	var fullText *string

	switch newStatus {
	case "DONE":
		// Получаем текст
		recognitionResults, err := w.saluteSpeech.GetRecognitionResult(taskResult.ResponseFileID)
		if err != nil {
			return fmt.Errorf("failed to get recognition result: %w", err)
		}

		text := recognitionResults.GetFullNormalizedText()
		fullText = &text

	case "ERROR", "CANCELED":
		// Оставим fullText как nil
		log.Printf("TranscriptionWorker: task %s failed with status %s", t.SaluteTaskID, newStatus)
	}

	// Обновляем в БД
	err = w.repo.Update(ctx, t.ID, newStatus, fullText)
	if err != nil {
		return fmt.Errorf("failed to update transcription in DB: %w", err)
	}

	log.Printf("TranscriptionWorker: updated transcription %s → Status=%s, HasText=%v", t.ID, newStatus, fullText != nil)

	return nil
}
