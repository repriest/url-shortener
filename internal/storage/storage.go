package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type URLEntry struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Storage interface {
	load() ([]URLEntry, error)
	append(entry URLEntry) error
}

type fileStorage struct {
	filePath string
}

func NewFileStorage(filePath string) Storage {
	return &fileStorage{filePath: filePath}
}

func (s *fileStorage) load() ([]URLEntry, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []URLEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	// parse file to []URLentry
	var entries []URLEntry
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := URLEntry{}
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			return []URLEntry{}, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *fileStorage) append(entry URLEntry) error {
	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", s.filePath, err)
	}
	defer file.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", s.filePath, err)
	}
	_, err = file.WriteString("\n")
	if err != nil {
		return fmt.Errorf("failed to write newline to file %s: %w", s.filePath, err)
	}

	return nil
}

type Repository struct {
	entries     []URLEntry
	uuidCounter int
	storage     Storage
}

func NewRepository(filePath string) (*Repository, error) {
	fs := NewFileStorage(filePath)
	repo := &Repository{
		storage:     fs,
		uuidCounter: 1,
	}
	entries, err := fs.load()
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

	entry := URLEntry{
		UUID:        idStr,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	r.entries = append(r.entries, entry)

	if err := r.storage.append(entry); err != nil {
		return fmt.Errorf("failed to add entry to storage: %w", err)
	}
	return nil
}
