// internal/repository/meeting_repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/google/uuid"
)

type MeetingRepository struct {
	db *DB
}

func NewMeetingRepository(db *DB) *MeetingRepository {
	return &MeetingRepository{db: db}
}

func (r *MeetingRepository) Create(ctx context.Context, meeting *models.Meeting) error {
	const query = `
		INSERT INTO meetings (
			id,
			user_id,
			title,
			audio_file_path,
			status,
			error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Conn.ExecContext(ctx, query,
		meeting.ID,
		meeting.UserID,
		meeting.Title,
		meeting.AudioFilePath,
		meeting.Status,
		meeting.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to create meeting: %w", err)
	}

	return nil
}

// UpdateMeeting обновляет существующую встречу
func (r *MeetingRepository) UpdateMeeting(ctx context.Context, meeting *models.Meeting) error {
	const query = `
		UPDATE meetings
		SET 
			title = $1,
			audio_file_path = $2,
			status = $3,
			error_message = $4
		WHERE id = $5
	`

	_, err := r.db.Conn.ExecContext(ctx, query,
		meeting.Title,
		meeting.AudioFilePath,
		meeting.Status,
		meeting.ErrorMessage,
		meeting.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update meeting: %w", err)
	}

	return nil
}

// UpdateStatusWithError обновляет статус встречи и опционально сообщение об ошибке.
// Если errorMessage пуст, в поле error_message будет записано NULL.
func (r *MeetingRepository) UpdateStatusWithError(ctx context.Context, id uuid.UUID, status, errorMessage string) error {
	const query = `
		UPDATE meetings
		SET status = $2, error_message = $3
		WHERE id = $1
	`

	var errorMsg *string
	if errorMessage != "" {
		errorMsg = &errorMessage
	}
	// Если errorMessage пустая — остаётся nil, в БД будет NULL
	_, err := r.db.Conn.ExecContext(ctx, query, id, status, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to update meeting status and error: %w", err)
	}

	return nil
}

// GetByUserAndID получает встречу по ID и пользователю, с транскрипцией и выжимкой
func (r *MeetingRepository) GetByUserAndID(ctx context.Context, userID int64, meetingID uuid.UUID) (*models.Meeting, error) {
	const query = `
		SELECT 
			m.id, m.user_id, m.title, m.audio_file_path, m.status, m.created_at,
			t.full_text AS transcription_text,
			s.summary_text AS summary_text
		FROM meetings m
		LEFT JOIN transcriptions t ON t.meeting_id = m.id
		LEFT JOIN summaries s ON s.meeting_id = m.id
		WHERE m.id = $1 AND m.user_id = $2
		LIMIT 1
	`

	row := r.db.Conn.QueryRowContext(ctx, query, meetingID, userID)

	var m models.Meeting
	var transcriptionText sql.NullString
	var summaryText sql.NullString

	err := row.Scan(
		&m.ID,
		&m.UserID,
		&m.Title,
		&m.AudioFilePath,
		&m.Status,
		&m.CreatedAt,
		&transcriptionText,
		&summaryText,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan meeting: %w", err)
	}

	if transcriptionText.Valid {
		m.TranscriptionText = &transcriptionText.String
	}
	if summaryText.Valid {
		m.SummaryText = &summaryText.String
	}

	return &m, nil
}

// ListByUser получает список встреч пользователя (с краткой информацией)
func (r *MeetingRepository) ListByUser(ctx context.Context, userID int64) ([]*models.Meeting, error) {
	const query = `
		SELECT id, user_id, title, status, created_at, audio_file_path
		FROM meetings
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Conn.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query meetings: %w", err)
	}
	defer rows.Close()

	var meetings []*models.Meeting

	for rows.Next() {
		var m models.Meeting
		err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.Title,
			&m.Status,
			&m.CreatedAt,
			&m.AudioFilePath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan meeting row: %w", err)
		}
		meetings = append(meetings, &m)
	}

	return meetings, nil
}

// SearchByUser ищет встречи по названию, транскрипции или выжимке
func (r *MeetingRepository) SearchByUser(ctx context.Context, userID int64, query string) ([]*models.Meeting, error) {
	const baseQuery = `
		SELECT 
			m.id, m.user_id, m.title, m.status, m.created_at,
			t.full_text AS transcription_text,
			s.summary_text AS summary_text
		FROM meetings m
		LEFT JOIN transcriptions t ON t.meeting_id = m.id
		LEFT JOIN summaries s ON s.meeting_id = m.id
		WHERE m.user_id = $1
		AND (
			m.title ILIKE $2
			OR t.full_text ILIKE $2
			OR s.summary_text ILIKE $2
		)
		ORDER BY m.created_at DESC
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.Conn.QueryContext(ctx, baseQuery, userID, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var meetings []*models.Meeting

	for rows.Next() {
		var m models.Meeting
		var transcriptionText sql.NullString
		var summaryText sql.NullString

		err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.Title,
			&m.Status,
			&m.CreatedAt,
			&transcriptionText,
			&summaryText,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan meeting row: %w", err)
		}

		if transcriptionText.Valid {
			m.TranscriptionText = &transcriptionText.String
		}
		if summaryText.Valid {
			m.SummaryText = &summaryText.String
		}

		meetings = append(meetings, &m)
	}

	return meetings, nil
}

// GetByMeetingID возвращает встречу по ID (без проверки пользователя)
func (r *MeetingRepository) GetByMeetingID(ctx context.Context, id uuid.UUID) (*models.Meeting, error) {
	const query = `
		SELECT 
			m.id, m.user_id, m.title, m.audio_file_path, m.status, m.created_at,
			t.full_text AS transcription_text,
			s.summary_text AS summary_text
		FROM meetings m
		LEFT JOIN transcriptions t ON t.meeting_id = m.id
		LEFT JOIN summaries s ON s.meeting_id = m.id
		WHERE m.id = $1
		LIMIT 1
	`

	row := r.db.Conn.QueryRowContext(ctx, query, id)

	var m models.Meeting
	var transcriptionText sql.NullString
	var summaryText sql.NullString

	err := row.Scan(
		&m.ID,
		&m.UserID,
		&m.Title,
		&m.AudioFilePath,
		&m.Status,
		&m.CreatedAt,
		&transcriptionText,
		&summaryText,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan meeting by ID %s: %w", id, err)
	}

	if transcriptionText.Valid {
		m.TranscriptionText = &transcriptionText.String
	}
	if summaryText.Valid {
		m.SummaryText = &summaryText.String
	}

	return &m, nil
}
