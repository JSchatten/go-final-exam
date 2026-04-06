// internal/worker/transcription_worker.go
package worker

import (
	"context"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/JSchatten/go-final-exam/internal/repository"
)

// TranscriptionWorker проверяет статус задач распознавания и обновляет статус транскрипции в БД.
type TranscriptionWorker struct {
	repo         *repository.TranscriptionRepository
	saluteSpeech *salutespeech.SaluteSpeechClient
	ticker       *time.Ticker
}

// NewTranscriptionWorker создаёт новый воркер.
func NewTranscriptionWorker(
	repo *repository.TranscriptionRepository,
	saluteSpeech *salutespeech.SaluteSpeechClient,
) *TranscriptionWorker {
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

// syncTranscription проверяет статус одной транскрипции и обновляет её в БД
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

	case "ERROR", "CANCELED":
		log.Printf("TranscriptionWorker: task %s failed with status %s", t.SaluteTaskID, newStatus)
		// Оставляем fullText = nil
	}

	// Обновляем ТОЛЬКО транскрипцию
	err = w.repo.Update(ctx, t.ID, newStatus, fullText)
	if err != nil {
		return err
	}

	log.Printf("TranscriptionWorker: updated transcription %v", t)
	return nil
}
