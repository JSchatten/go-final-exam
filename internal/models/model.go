package models

import (
	"time"

	"github.com/google/uuid"
)

// Transcription - результат распознавания речи (SaluteSpeech)
type Transcription struct {
	ID          uuid.UUID `json:"id" db:"id"`
	MeetingID   uuid.UUID `json:"meeting_id" db:"meeting_id"`
	FullText    string    `json:"full_text" db:"full_text"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
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
