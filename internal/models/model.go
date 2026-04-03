package models

import (
	"time"

	"github.com/google/uuid"
)

// User - пользователь Telegram
type User struct {
	ID         int64     `json:"id" db:"id"`                   // внутренний ID (BIGSERIAL)
	TelegramID int64     `json:"telegram_id" db:"telegram_id"` // настоящий Telegram ID
	Username   *string   `json:"username,omitempty" db:"username"`
	FirstName  *string   `json:"first_name,omitempty" db:"first_name"`
	LastName   *string   `json:"last_name,omitempty" db:"last_name"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Meeting - встреча (загруженная аудиозапись)
type Meeting struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          int64      `json:"user_id" db:"user_id"` // ссылается на telegram_id из users
	Title           string     `json:"title" db:"title"`
	AudioFilePath   *string    `json:"audio_file_path,omitempty" db:"audio_file_path"`
	Status          string     `json:"status" db:"status"` // uploaded, processing, completed, failed
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	TranscriptionID *uuid.UUID `json:"transcription_id,omitempty" db:"transcription_id"`
	SummaryID       *uuid.UUID `json:"summary_id,omitempty" db:"summary_id"`
}

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
