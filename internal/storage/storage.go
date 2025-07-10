package storage

import (
	"fmt"
	"github.com/repriest/url-shortener/internal/storage/file"
	"github.com/repriest/url-shortener/internal/storage/memory"
	"github.com/repriest/url-shortener/internal/storage/postgres"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"strconv"
)

type Repository struct {
	uuidCounter int
	storage     t.Storage
}

func NewPostgresStorage(dsn string) (t.Storage, error) {
	return postgres.NewPgStorage(dsn)
}

func NewFileStorage(filePath string) (t.Storage, error) {
	return file.NewFileStorage(filePath)
}

func NewMemoryStorage() t.Storage {
	return memory.NewMemoryStorage()
}

func NewRepository(st t.Storage) (*Repository, error) {
	repo := &Repository{
		storage:     st,
		uuidCounter: 1,
	}
	entries, err := st.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load entries: %w", err)
	}
	for _, entry := range entries {
		id, err := strconv.Atoi(entry.UUID)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID: %s", entry.UUID)
		}
		// set counter for next record
		if id >= repo.uuidCounter {
			repo.uuidCounter = id + 1
		}
	}
	return repo, nil
}

func (r *Repository) AddNewEntry(shortURL string, originalURL string) error {
	idStr := strconv.Itoa(r.uuidCounter)
	r.uuidCounter++

	entry := t.URLEntry{
		UUID:        idStr,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}

	return r.storage.Append(entry)
}

func (r *Repository) Close() error {
	return r.storage.Close()
}

func (r *Repository) BatchAppend(entries []t.URLEntry) error {
	return r.storage.BatchAppend(entries)
}
