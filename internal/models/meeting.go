package models

import (
	"time"

	"github.com/google/uuid"
)

type Meeting struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          int64      `json:"user_id" db:"user_id"`
	Title           string     `json:"title" db:"title"`
	AudioFilePath   *string    `json:"audio_file_path,omitempty" db:"audio_file_path"`
	Status          string     `json:"status" db:"status"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	TranscriptionID *uuid.UUID `json:"transcription_id,omitempty" db:"transcription_id"`
	SummaryID       *uuid.UUID `json:"summary_id,omitempty" db:"summary_id"`

	// Дополнительные поля для вывода (не сохраняются в meetings)
	TranscriptionText *string `json:"transcription_text,omitempty"`
	SummaryText       *string `json:"summary_text,omitempty"`
}
