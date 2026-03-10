package sqllite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Connect подключается к SQLite базе данных
func Connect(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

// ConnectInMemory создаёт базу данных в памяти
func ConnectInMemory() (*sql.DB, error) {
	return Connect(":memory:")
}
