package storage

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/repriest/url-shortener/internal/storage/file"
	"github.com/repriest/url-shortener/internal/storage/memory"
	"github.com/repriest/url-shortener/internal/storage/postgres"
	t "github.com/repriest/url-shortener/internal/storage/types"
)

type Repository struct {
	storage t.Storage
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
		storage: st,
	}
	_, err := st.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load entries: %w", err)
	}
	return repo, nil
}

func (r *Repository) AddNewEntry(shortURL string, originalURL string) error {
	idStr := uuid.New().String()

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
	for i := range entries {
		entries[i].UUID = uuid.New().String()
	}
	return r.storage.BatchAppend(entries)
}
