package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/repriest/url-shortener/internal/contextkeys"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"net/http"
	"time"
)

func (h *Handler) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	longURL, err := getLongURL(r)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if longURL == "" {
		w.WriteHeader(http.StatusCreated)
		return
	}

	// shorten URL
	shortURL, err := shortenURL(longURL)
	if err != nil {
		http.Error(w, "Could not shorten URL", http.StatusBadRequest)
		return
	}
	responseURL := h.cfg.BaseURL + "/" + shortURL

	userIDVal := r.Context().Value(contextkeys.UserIDKey)
	userID, ok := userIDVal.(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	entry := t.URLEntry{
		UUID:        uuid.New().String(),
		ShortURL:    shortURL,
		OriginalURL: longURL,
		UserID:      userID,
	}

	// check existing shortURL
	err = h.st.Append(entry)
	if err != nil {
		var urlConflictError *t.URLConflictError
		if errors.As(err, &urlConflictError) { // get instance of URLConflictError if err matches
			if urlConflictError.ShortURL != "" {
				responseURL = h.cfg.BaseURL + "/" + urlConflictError.ShortURL
			}
			// write existing shortened URL
			w.WriteHeader(http.StatusConflict)
			if _, err := w.Write([]byte(responseURL)); err != nil {
				http.Error(w, "Could not write URL", http.StatusInternalServerError)
				return
			}
			return
		}
		http.Error(w, "Could not write URL to storage", http.StatusInternalServerError)
		return
	}

	// write response
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(responseURL)); err != nil {
		http.Error(w, "Could not write URL", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ExpandHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")
	longURL, err := expandURL(shortURL)
	if err != nil {
		http.Error(w, "Could not decode URL", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)
}

func (h *Handler) ShortenJSONHandler(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest

	// read body
	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// parse body json
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// shorten URL
	shortURL, err := shortenURL(req.URL)
	if err != nil {
		http.Error(w, "Could not shorten URL", http.StatusBadRequest)
		return
	}
	responseURL := ShortenResponse{h.cfg.BaseURL + "/" + shortURL}

	userIDVal := r.Context().Value(contextkeys.UserIDKey)
	userID, ok := userIDVal.(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	entry := t.URLEntry{
		UUID:        uuid.New().String(),
		ShortURL:    shortURL,
		OriginalURL: req.URL,
		UserID:      userID,
	}

	// check existing shortURL
	err = h.st.Append(entry)
	if err != nil {
		var urlConflictError *t.URLConflictError
		if errors.As(err, &urlConflictError) { // get instance of URLConflictError if err matches
			if urlConflictError.ShortURL != "" {
				responseURL = ShortenResponse{h.cfg.BaseURL + "/" + urlConflictError.ShortURL}
			}
			// write existing shortened URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			err := json.NewEncoder(w).Encode(responseURL)
			if err != nil {
				http.Error(w, "Could not encode response", http.StatusInternalServerError)
				return
			}
			return
		}
		http.Error(w, "Could not write URL to storage", http.StatusInternalServerError)
		return
	}

	// write shortened URL
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(responseURL)
	if err != nil {
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if h.cfg.DatabaseDSN == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.st.Ping(ctx)
	if err != nil {
		http.Error(w, "Failed to ping database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	var req []ShortenBatchRequest
	var resp []ShortenBatchResponse

	// read body
	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	var entries []t.URLEntry
	userIDVal := r.Context().Value(contextkeys.UserIDKey)
	userID, ok := userIDVal.(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// parese entries
	for _, reqEntry := range req {
		if reqEntry.OriginalURL == "" {
			http.Error(w, "Empty URL", http.StatusBadRequest)
			return
		}
		shortURL, err := shortenURL(reqEntry.OriginalURL)
		if err != nil {
			http.Error(w, "Could not shorten URL", http.StatusBadRequest)
			return
		}
		entry := t.URLEntry{
			UUID:        reqEntry.CorrelationID,
			ShortURL:    shortURL,
			OriginalURL: reqEntry.OriginalURL,
			UserID:      userID,
		}
		entries = append(entries, entry)
		resp = append(resp, ShortenBatchResponse{
			CorrelationID: reqEntry.CorrelationID,
			ShortURL:      h.cfg.BaseURL + "/" + shortURL,
		})
	}

	err = h.st.BatchAppend(entries)
	if err != nil {
		http.Error(w, "Failed to batch append", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value(contextkeys.UserIDKey)
	userID, ok := userIDVal.(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	urls, err := h.st.GetByUserID(userID)
	if err != nil {
		http.Error(w, "Could not get URLs", http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	response := make([]map[string]string, len(urls))
	for i, urlEntry := range urls {
		response[i] = map[string]string{
			"short_url":    h.cfg.BaseURL + "/" + urlEntry.ShortURL,
			"original_url": urlEntry.OriginalURL,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
	}
}
