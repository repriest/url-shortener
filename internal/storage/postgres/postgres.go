package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"time"
)

type PGStorage struct {
	db         *sql.DB
	deleteChan chan deleteRequest
}

type deleteRequest struct {
	UserID    string
	ShortURLs []string
}

func NewPgStorage(dsn string) (*PGStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			uuid TEXT PRIMARY KEY,
			short_url TEXT NOT NULL,
			original_url TEXT NOT NULL UNIQUE,
			user_id TEXT NOT NULL,
			is_deleted BOOLEAN DEFAULT FALSE
		)
	`)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table urls: %w", err)
	}

	s := PGStorage{db: db, deleteChan: make(chan deleteRequest, 100)}

	go s.asyncDelete()

	return &PGStorage{db: db}, nil
}

func (s *PGStorage) asyncDelete() {
	for req := range s.deleteChan {
		if len(req.ShortURLs) == 0 {
			continue
		}
		_, err := s.db.Exec(`
			UPDATE urls 
			SET is_deleted = TRUE 
			WHERE user_id = $1 AND short_url = ANY($2)`, req.UserID, req.ShortURLs)
		if err != nil {
			fmt.Printf("Failed to delete urls: %v\n", err)
		}
	}
}

func (s *PGStorage) QueueDelete(userID string, shortURLs []string) {
	s.deleteChan <- deleteRequest{UserID: userID, ShortURLs: shortURLs}
}

func (s *PGStorage) Load() ([]t.URLEntry, error) {
	rows, err := s.db.Query("SELECT uuid, short_url, original_url, user_id, is_deleted FROM urls")
	if err != nil {
		return nil, fmt.Errorf("failed to query urls: %w", err)
	}
	defer rows.Close()

	var entries []t.URLEntry
	for rows.Next() {
		entry := t.URLEntry{}
		err := rows.Scan(&entry.UUID, &entry.ShortURL, &entry.OriginalURL, &entry.UserID, &entry.IsDeleted)
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

func (s *PGStorage) Append(entry t.URLEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// try to insert entry
	result, err := s.db.ExecContext(ctx, `
			INSERT INTO urls (uuid, short_url, original_url, user_id, is_deleted) 
			VALUES ($1, $2, $3, $4, FALSE)
			ON CONFLICT (original_url) DO NOTHING 
		`, entry.UUID, entry.ShortURL, entry.OriginalURL, entry.UserID)
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

func (s *PGStorage) BatchAppend(entries []t.URLEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// prepare insert entry statement
	stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO urls (uuid, short_url, original_url, user_id, is_deleted) 
			VALUES ($1, $2, $3, $4, FALSE)
			ON CONFLICT (original_url) DO NOTHING 
		`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// execute insert entry statement
	for _, entry := range entries {
		_, err = stmt.ExecContext(ctx, entry.UUID, entry.ShortURL, entry.OriginalURL, entry.UserID)
		if err != nil {
			return fmt.Errorf("failed to insert entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *PGStorage) Close() error {
	return s.db.Close()
}

func (s *PGStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *PGStorage) GetByUserID(userID string) ([]t.URLEntry, error) {
	rows, err := s.db.Query(`
		SELECT uuid, short_url, original_url, user_id, is_deleted
		FROM urls
		WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query urls by user_id: %w", err)
	}
	defer rows.Close()

	var entries []t.URLEntry
	for rows.Next() {
		entry := t.URLEntry{}
		err := rows.Scan(&entry.UUID, &entry.ShortURL, &entry.OriginalURL, &entry.UserID, &entry.IsDeleted)
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

func (s *PGStorage) GetByShortURL(shortURL string) (*t.URLEntry, error) {
	row := s.db.QueryRow(`
		SELECT uuid, short_url, original_url, user_id, is_deleted 
		FROM urls 
		WHERE short_url = $1`, shortURL)
	var entry t.URLEntry
	err := row.Scan(&entry.UUID, &entry.ShortURL, &entry.OriginalURL, &entry.UserID, &entry.IsDeleted)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	return &entry, nil
}
