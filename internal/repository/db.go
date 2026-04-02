// internal/repository/db.go
package repository

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB holds the database connection
type DB struct {
	Conn *sql.DB
}

// InitDB initializes the database connection using Config and runs migrations
func InitDB(cfg *Config) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config is required")
	}

	// Формируем DSN из конфига
	dataSourceName := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name,
	)

	// Открываем соединение через pgx (используя stdlib-обёртку)
	conn, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем подключение
	if err = conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established")

	// Подготавливаем экземпляр базы для migrate
	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("could not create migrate driver: %w", err)
	}

	// Создаём мигратор
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations", // путь к папке с миграциями
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create migrate instance: %w", err)
	}

	// Выполняем миграции "вперёд"
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("could not run migrate up: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("Migrations applied: no changes (already up to date)")
	} else {
		log.Println("Migrations applied successfully")
	}

	return &DB{Conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	if db.Conn != nil {
		err := db.Conn.Close()
		if err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}
