package handlers

import (
	"errors"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"io"
	"net/http"
)

func readRequestBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

func writeResponse(w http.ResponseWriter, content []byte) {
	if _, err := w.Write(content); err != nil {
		http.Error(w, "Could not write response", http.StatusInternalServerError)
	}
}

func handleStorageError(w http.ResponseWriter, err error) {
	if errors.Is(err, t.ErrURLExists) {
		w.WriteHeader(http.StatusConflict)
	} else {
		http.Error(w, "Could not write URL to storage", http.StatusInternalServerError)
	}
}

func getLongURL(r *http.Request) (string, error) {
	body, err := readRequestBody(r)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
