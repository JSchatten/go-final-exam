// internal/repository/transcription_repository.go
package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TranscriptionRepository struct {
	db *DB
}

func NewTranscriptionRepository(db *DB) *TranscriptionRepository {
	return &TranscriptionRepository{db: db}
}

func (r *TranscriptionRepository) Create(meetingID uuid.UUID, text string) (uuid.UUID, error) {
	id := uuid.New()
	const query = `
		INSERT INTO transcriptions (id, meeting_id, full_text, processed_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Conn.Exec(query, id, meetingID, text, time.Now().UTC())
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create transcription: %w", err)
	}

	return id, nil
}
