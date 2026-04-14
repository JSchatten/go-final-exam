// internal/repository/user_repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/JSchatten/go-final-exam/internal/models"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateIfNotExists(ctx context.Context, user *models.User) error {
	const query = `
		INSERT INTO users (telegram_id, username, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (telegram_id) DO NOTHING
	`

	_, err := r.db.Conn.ExecContext(ctx, query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
	)

	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (r *UserRepository) FindByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	const query = `
		SELECT id, telegram_id, username, first_name, last_name, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
		LIMIT 1
	`

	row := r.db.Conn.QueryRowContext(ctx, query, telegramID)
	var u models.User

	err := u.ScanRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return &u, nil
}

// Update updates the users updated_at timestamp and other fields
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	const query = `
		UPDATE users 
		SET username = $2, first_name = $3, last_name = $4, updated_at = $5
		WHERE telegram_id = $1
	`

	_, err := r.db.Conn.ExecContext(ctx, query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
