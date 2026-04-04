// internal/models/user.go
package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID         int64     `json:"id" db:"id"`
	TelegramID int64     `json:"telegram_id" db:"telegram_id"`
	Username   *string   `json:"username,omitempty" db:"username"`
	FirstName  *string   `json:"first_name,omitempty" db:"first_name"`
	LastName   *string   `json:"last_name,omitempty" db:"last_name"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

func (u *User) TableName() string {
	return "users"
}

func (u *User) GetID() any {
	return u.ID
}

func (u *User) SetID(id any) {
	if i, ok := id.(int64); ok {
		u.ID = i
	}
}

// ScanRow сканирует строку из SQL-запроса в структуру User
func (u *User) ScanRow(row *sql.Row) error {
	return row.Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.FirstName,
		&u.LastName,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
}
