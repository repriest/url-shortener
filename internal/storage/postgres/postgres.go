package postgres

import (
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	t "github.com/repriest/url-shortener/internal/storage/types"
)

type pgStorage struct {
	db *sql.DB
}

func NewPgStorage(dsn string) (t.Storage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			uuid TEXT PRIMARY KEY,
			short_url TEXT NOT NULL,
			original_url TEXT NOT NULL
		)
	`)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table urls: %w", err)
	}

	return &pgStorage{db: db}, nil
}

func (s pgStorage) Load() ([]t.URLEntry, error) {
	rows, err := s.db.Query("SELECT uuid, short_url, original_url FROM urls")
	if err != nil {
		return nil, fmt.Errorf("failed to query urls: %w", err)
	}
	defer rows.Close()

	var entries []t.URLEntry
	for rows.Next() {
		entry := t.URLEntry{}
		err := rows.Scan(&entry.UUID, &entry.ShortURL, &entry.OriginalURL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		entries = append(entries, entry)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}
	return entries, nil
}

func (s pgStorage) Append(entry t.URLEntry) error {
	_, err := s.db.Exec(`
		INSERT INTO urls (uuid, short_url, original_url) 
		VALUES ($1, $2, $3)
	`, entry.UUID, entry.ShortURL, entry.OriginalURL)

	if err != nil {
		return fmt.Errorf("failed to insert url: %w", err)
	}
	return nil
}

func (s pgStorage) Close() error {
	return s.db.Close()
}
