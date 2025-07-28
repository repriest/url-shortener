package handlers

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
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

func getLongURL(r *http.Request) (string, error) {
	body, err := readRequestBody(r)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func shortenURL(longURL string) (string, error) {
	if _, err := url.ParseRequestURI(longURL); err != nil {
		return "", err
	}
	shortURL := base64.StdEncoding.EncodeToString([]byte(longURL))
	return shortURL, nil
}

func expandURL(shortURL string) (string, error) {
	longURLBytes, err := base64.StdEncoding.DecodeString(shortURL)
	if err != nil {
		return "", err
	}
	longURL := string(longURLBytes)
	if _, err := url.ParseRequestURI(longURL); err != nil {
		return "", err
	}
	return longURL, nil
}
