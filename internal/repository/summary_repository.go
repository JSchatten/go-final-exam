// internal/repository/summary_repository.go
package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SummaryRepository struct {
	db *DB
}

func NewSummaryRepository(db *DB) *SummaryRepository {
	return &SummaryRepository{db: db}
}

func (r *SummaryRepository) Create(meetingID uuid.UUID, text string) (uuid.UUID, error) {
	id := uuid.New()
	const query = `
		INSERT INTO summaries (id, meeting_id, summary_text, generated_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Conn.Exec(query, id, meetingID, text, time.Now().UTC())
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create summary: %w", err)
	}

	return id, nil
}
