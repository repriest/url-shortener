package types

import "errors"

var ErrURLExists = errors.New("URL already exists")

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
