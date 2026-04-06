package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const (
	StatusUploaded   = "uploaded"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Transcription - результат распознавания речи (SaluteSpeech)
type Transcription struct {
	ID           uuid.UUID `json:"id" db:"id"`
	MeetingID    uuid.UUID `json:"meeting_id" db:"meeting_id"`
	SaluteTaskID string    `json:"salute_task_id" db:"salute_task_id"`
	Status       string    `json:"status" db:"status"`
	FullText     string    `json:"full_text" db:"full_text"`
	ProcessedAt  time.Time `json:"processed_at" db:"processed_at"`
}

// Summary - краткая выжимка от GigaChat
type Summary struct {
	ID          uuid.UUID `json:"id" db:"id"`
	MeetingID   uuid.UUID `json:"meeting_id" db:"meeting_id"`
	SummaryText string    `json:"summary_text" db:"summary_text"`
	GeneratedAt time.Time `json:"generated_at" db:"generated_at"`
}

// ChatHistory - история диалогов с GigaChat (/chat)
type ChatHistory struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       int64     `json:"user_id" db:"user_id"` // telegram_id
	QueryText    string    `json:"query_text" db:"query_text"`
	ResponseText string    `json:"response_text" db:"response_text"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Meeting struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	UserID        int64          `json:"user_id" db:"user_id"`
	Title         string         `json:"title" db:"title"`
	AudioFilePath *string        `json:"audio_file_path,omitempty" db:"audio_file_path"`
	Status        string         `json:"status" db:"status"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	ErrorMessage  sql.NullString `json:"error_message,omitempty" db:"error_message"`

	// Дополнительные поля для вывода (не сохраняются в meetings)
	TranscriptionText *string `json:"transcription_text,omitempty"`
	SummaryText       *string `json:"summary_text,omitempty"`
}
