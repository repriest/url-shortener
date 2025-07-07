package memory

import (
	t "github.com/repriest/url-shortener/internal/storage/types"
)

type memoryStorage struct {
	entries []t.URLEntry
}

func NewMemoryStorage() t.Storage {
	return &memoryStorage{
		entries: []t.URLEntry{},
	}
}

func (s *memoryStorage) Load() ([]t.URLEntry, error) {
	return s.entries, nil
}

func (s *memoryStorage) Append(entry t.URLEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *memoryStorage) Close() error {
	return nil
}
