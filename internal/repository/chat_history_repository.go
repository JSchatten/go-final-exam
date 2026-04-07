package repository

import (
	"context"
	"fmt"

	"github.com/JSchatten/go-final-exam/internal/models"
)

// ChatHistoryRepository предоставляет методы для работы с историей чата.
type ChatHistoryRepository struct {
	db *DB
}

// NewChatHistoryRepository создаёт новый экземпляр репозитория.
func NewChatHistoryRepository(db *DB) *ChatHistoryRepository {
	return &ChatHistoryRepository{db: db}
}

// Create сохраняет новую запись в истории чата.
// Поля id и created_at заполняются автоматически на стороне БД.
func (r *ChatHistoryRepository) Create(ctx context.Context, userID int64, queryText, responseText string) error {
	const query = `
		INSERT INTO chat_history (user_id, query_text, response_text)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Conn.ExecContext(ctx, query, userID, queryText, responseText)
	if err != nil {
		return fmt.Errorf("failed to insert chat history: %w", err)
	}

	return nil
}

// ListByUser возвращает последние N записей истории чата для пользователя.
func (r *ChatHistoryRepository) ListByUser(ctx context.Context, userID int64, limit int) ([]*models.ChatHistory, error) {
	const query = `
		SELECT id, user_id, query_text, response_text, created_at
		FROM chat_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Conn.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat history: %w", err)
	}
	defer rows.Close()

	var history []*models.ChatHistory
	for rows.Next() {
		var record models.ChatHistory
		err := rows.Scan(
			&record.ID,
			&record.UserID,
			&record.QueryText,
			&record.ResponseText,
			&record.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat history row: %w", err)
		}
		history = append(history, &record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over chat history rows: %w", err)
	}

	return history, nil
}
