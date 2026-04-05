package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/google/uuid"
)

type TranscriptionRepository struct {
	db *DB
}

func NewTranscriptionRepository(db *DB) *TranscriptionRepository {
	return &TranscriptionRepository{db: db}
}

// Create создает новую запись транскрипции с начальным статусом и ID задачи SaluteSpeech
func (r *TranscriptionRepository) Create(meetingID uuid.UUID, saluteTaskID string, status string) (uuid.UUID, error) {
	id := uuid.New()
	const query = `
		INSERT INTO transcriptions (id, meeting_id, salute_task_id, status, processed_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Conn.Exec(query, id, meetingID, saluteTaskID, status, time.Now().UTC())
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create transcription: %w", err)
	}

	return id, nil
}

// Update обновляет транскрипцию — статус, текст и время обработки
// Используется при завершении распознавания
func (r *TranscriptionRepository) Update(id uuid.UUID, status, fullText string) error {
	const query = `
		UPDATE transcriptions
		SET status = $1, full_text = $2, processed_at = $3
		WHERE id = $4
	`

	_, err := r.db.Conn.Exec(query, status, fullText, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update transcription: %w", err)
	}

	return nil
}

// GetByMeetingID возвращает транскрипцию по ID встречи
func (r *TranscriptionRepository) GetByMeetingID(meetingID uuid.UUID) (*models.Transcription, error) {
	const query = `
		SELECT id, meeting_id, salute_task_id, status, full_text, processed_at
		FROM transcriptions
		WHERE meeting_id = $1
		LIMIT 1
	`

	var transcription models.Transcription
	err := r.db.Conn.QueryRow(query, meetingID).Scan(
		&transcription.ID,
		&transcription.MeetingID,
		&transcription.SaluteTaskID,
		&transcription.Status,
		&transcription.FullText,
		&transcription.ProcessedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transcription not found for meeting %s", meetingID)
		}
		return nil, fmt.Errorf("failed to get transcription from DB: %w", err)
	}

	return &transcription, nil
}

// GetByID возвращает транскрипцию по ID
func (r *TranscriptionRepository) GetByID(id uuid.UUID) (*models.Transcription, error) {
	const query = `
		SELECT id, meeting_id, salute_task_id, status, full_text, processed_at
		FROM transcriptions
		WHERE id = $1
		LIMIT 1
	`

	var transcription models.Transcription
	err := r.db.Conn.QueryRow(query, id).Scan(
		&transcription.ID,
		&transcription.MeetingID,
		&transcription.SaluteTaskID,
		&transcription.Status,
		&transcription.FullText,
		&transcription.ProcessedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transcription with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get transcription by ID: %w", err)
	}

	return &transcription, nil
}
