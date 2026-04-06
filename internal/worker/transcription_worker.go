// internal/worker/transcription_worker.go
package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/repository"
)

// TranscriptionWorker опрашивает необработанные транскрипции и логирует их.
type TranscriptionWorker struct {
	repo   *repository.TranscriptionRepository
	ticker *time.Ticker
}

// NewTranscriptionWorker создаёт новый воркер с репозиторием.
func NewTranscriptionWorker(repo *repository.TranscriptionRepository) *TranscriptionWorker {
	return &TranscriptionWorker{
		repo:   repo,
		ticker: time.NewTicker(5 * time.Second),
	}
}

// Start запускает фоновый опрос.
func (w *TranscriptionWorker) Start(ctx context.Context) error {
	log.Println("TranscriptionWorker: started polling UNPROCESSED transcriptions every 5 seconds")
	defer w.ticker.Stop()

	for {
		select {
		case <-w.ticker.C:
			if err := w.pollUnprocessed(ctx); err != nil {
				log.Printf("TranscriptionWorker: failed to poll unprocessed transcriptions: %v", err)
			}
		case <-ctx.Done():
			log.Println("TranscriptionWorker: shutting down...")
			return ctx.Err()
		}
	}
}

// pollUnprocessed получает транскрипции со статусом NEW или RUNNING и логирует каждую
func (w *TranscriptionWorker) pollUnprocessed(ctx context.Context) error {
	transcriptions, err := w.repo.GetUnprocessed(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed transcriptions: %w", err)
	}

	log.Printf("TranscriptionWorker: found %d UNPROCESSED transcription(s)", len(transcriptions))

	for _, t := range transcriptions {
		log.Printf("транскрипция: %v", t)
	}

	return nil
}
