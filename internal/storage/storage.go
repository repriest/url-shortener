package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type URLEntry struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Storage struct {
	entries     []URLEntry
	filePath    string
	uuidCounter int
}

func NewStorage(filePath string) (*Storage, error) {
	s := &Storage{filePath: filePath, uuidCounter: 1}
	err := s.load()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) load() error {
	// open or create file
	file, err := os.OpenFile(s.filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// read file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	// if file is empty - init json with empty array
	if len(data) == 0 {
		s.entries = []URLEntry{}
		_, err = file.Write([]byte("[]"))
		if err != nil {
			return fmt.Errorf("failed to initialize empty file %s: %w", s.filePath, err)
		}
		return nil
	}

	err = json.Unmarshal(data, &s.entries)
	if err != nil {
		return err
	}

	// find max uuid in file
	for _, entry := range s.entries {
		id, err := strconv.Atoi(entry.UUID)
		if err != nil {
			return fmt.Errorf("invalid UUID: %s", entry.UUID)
		}
		// set counter for next record
		if id >= s.uuidCounter {
			s.uuidCounter = id + 1
		}
	}
	return nil
}

func (s *Storage) save() error {
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Storage) AddNewEntry(shortURL string, originalURL string) error {
	idStr := strconv.Itoa(s.uuidCounter)
	s.uuidCounter++

	entry := URLEntry{
		UUID:        idStr,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	s.entries = append(s.entries, entry)
	err := s.save()
	if err != nil {
		return err
	}
	return nil
}
