package db

import (
	"database/sql"
	"fmt"
	"log"
)

// DB holds the database connection
type DB struct {
	Conn *sql.DB
}

// InitDB initializes the database connection
func InitDB(dataSourceName string) (*DB, error) {
	conn, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	err = conn.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established")
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
