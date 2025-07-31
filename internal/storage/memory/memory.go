package memory

import (
	"context"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"slices"
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

func (s *MemoryStorage) GetByShortURL(shortURL string) (*t.URLEntry, error) {
	for _, entry := range s.entries {
		if entry.ShortURL == shortURL {
			return &entry, nil
		}
	}

	return nil, nil
}

func (s *MemoryStorage) QueueDelete(userID string, shortURLs []string) {
	for i := range s.entries {
		if s.entries[i].UserID == userID && slices.Contains(shortURLs, s.entries[i].ShortURL) {
			s.entries[i].IsDeleted = true
		}
	}
}
