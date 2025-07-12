package postgres

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"time"
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
			original_url TEXT NOT NULL UNIQUE
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// try to insert entry
	result, err := s.db.ExecContext(ctx, `
			INSERT INTO urls (uuid, short_url, original_url) 
			VALUES ($1, $2, $3)
			ON CONFLICT (original_url) DO NOTHING 
		`, entry.UUID, entry.ShortURL, entry.OriginalURL)
	if err != nil {
		return fmt.Errorf("failed to insert url: %w", err)
	}

	// check if url was inserted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// if nothing was inserted - return error with existing short url
	if rowsAffected == 0 {
		var existingShortURL string
		err := s.db.QueryRowContext(ctx, "SELECT short_url FROM urls WHERE original_url = $1", entry.OriginalURL).Scan(&existingShortURL)
		if err != nil {
			return fmt.Errorf("failed to query existing short url: %w", err)
		}
		return &t.URLConflictError{ShortURL: existingShortURL}
	}

	return nil
}

func (s pgStorage) Close() error {
	return s.db.Close()
}

func (s pgStorage) BatchAppend(entries []t.URLEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// prepare insert entry statement
	stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO urls (uuid, short_url, original_url) 
			VALUES ($1, $2, $3)
			ON CONFLICT (original_url) DO NOTHING 
		`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// execute insert entry statement
	for _, entry := range entries {
		_, err = stmt.ExecContext(ctx, entry.UUID, entry.ShortURL, entry.OriginalURL)
		if err != nil {
			return fmt.Errorf("failed to insert entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
