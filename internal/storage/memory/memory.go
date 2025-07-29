package memory

import (
	"context"
	t "github.com/repriest/url-shortener/internal/storage/types"
)

type MemoryStorage struct {
	entries []t.URLEntry
}

func NewMemoryStorage() (*MemoryStorage, error) {
	return &MemoryStorage{
		entries: []t.URLEntry{},
	}, nil
}

func (s *MemoryStorage) Load() ([]t.URLEntry, error) {
	return s.entries, nil
}

func (s *MemoryStorage) Append(entry t.URLEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *MemoryStorage) BatchAppend(entries []t.URLEntry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

func (s *MemoryStorage) Close() error {
	return nil
}

func (s *MemoryStorage) Ping(ctx context.Context) error {
	return nil
}

func (s *MemoryStorage) GetByUserID(userID string) ([]t.URLEntry, error) {
	var userEntries []t.URLEntry
	for _, entry := range s.entries {
		if entry.UserID == userID {
			userEntries = append(userEntries, entry)
		}
	}
	return userEntries, nil
}
