package types

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
