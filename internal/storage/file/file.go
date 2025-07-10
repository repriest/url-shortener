package file

import (
	"encoding/json"
	"fmt"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"os"
	"strings"
)

type fileStorage struct {
	file *os.File
}

func NewFileStorage(filePath string) (t.Storage, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open or create file: %w", err)
	}

	return &fileStorage{file: file}, nil
}

func (s *fileStorage) Load() ([]t.URLEntry, error) {
	data, err := os.ReadFile(s.file.Name())
	if err != nil {
		if os.IsNotExist(err) {
			return []t.URLEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	// parse file to []URLentry
	var entries []t.URLEntry
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := t.URLEntry{}
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			return []t.URLEntry{}, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *fileStorage) Append(entry t.URLEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}
	data = append(data, '\n')

	_, err = s.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", s.file.Name(), err)
	}

	return nil
}

func (s *fileStorage) Close() error {
	return s.file.Close()
}

func (s *fileStorage) BatchAppend(entries []t.URLEntry) error {
	var data []byte

	for _, entry := range entries {
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		data = append(data, entryJSON...)
		data = append(data, '\n')
	}

	_, err := s.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", s.file.Name(), err)
	}

	return nil
}
