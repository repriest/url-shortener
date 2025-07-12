package types

import (
	"errors"
	"fmt"
)

var ErrURLExists = errors.New("URL already exists")

type URLConflictError struct {
	ShortURL string
}

func (e *URLConflictError) Error() string {
	return fmt.Sprintf("URL already exists with shortURL: %s", e.ShortURL)
}

type URLEntry struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Storage interface {
	Load() ([]URLEntry, error)
	Append(entry URLEntry) error
	BatchAppend(entries []URLEntry) error
	Close() error
}
