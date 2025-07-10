package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/storage"
	t "github.com/repriest/url-shortener/internal/storage/types"
	"github.com/repriest/url-shortener/internal/urlservice"
	"io"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	cfg *config.Config
	st  *storage.Repository
}

func NewHandler(cfg *config.Config, st *storage.Repository) *Handler {
	return &Handler{cfg: cfg, st: st}
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

type ShortenBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortenBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"original_url"`
}

func readRequestBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (h *Handler) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	// read body
	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "Could not read body", http.StatusBadRequest)
		return
	}

	// check if URL is empty
	longURL := string(body)
	if longURL == "" {
		w.WriteHeader(http.StatusCreated)
		return
	}

	// shorten URL
	shortURL, err := urlservice.ShortenURL(longURL)
	if err != nil {
		http.Error(w, "Could not shorten URL", http.StatusBadRequest)
		return
	}

	// write shortened URL
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(h.cfg.BaseURL + "/" + shortURL)); err != nil {
		http.Error(w, "Could not write URL", http.StatusInternalServerError)
		return
	}

	// append entry uuid - shorturl - longurl to file
	err = h.st.AddNewEntry(shortURL, longURL)
	if err != nil {
		http.Error(w, "Could not write URL", http.StatusInternalServerError)
	}
}

func (h *Handler) ExpandHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")
	longURL, err := urlservice.ExpandURL(shortURL)
	if err != nil {
		http.Error(w, "Could not decode URL", http.StatusBadRequest)
	}
	log.Printf("short url: %s\n", h.cfg.BaseURL+"/"+shortURL)
	log.Printf("long url: %s\n", longURL)
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
	shortURL, err := urlservice.ShortenURL(req.URL)
	if err != nil {
		http.Error(w, "Could not shorten URL", http.StatusBadRequest)
		return
	}
	// write shortened URL
	resp := ShortenResponse{Result: h.cfg.BaseURL + "/" + shortURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	respJSON, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)

	// append entry uuid - shorturl - longurl to file
	err = h.st.AddNewEntry(shortURL, req.URL)
	if err != nil {
		http.Error(w, "Could not write URL", http.StatusInternalServerError)
	}
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if h.cfg.DatabaseDSN == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	db, err := sql.Open("pgx", h.cfg.DatabaseDSN)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
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

	// parese entries
	for _, reqEntry := range req {
		if reqEntry.OriginalURL == "" {
			http.Error(w, "Empty URL", http.StatusBadRequest)
			return
		}
		shortURL, err := urlservice.ShortenURL(reqEntry.OriginalURL)
		if err != nil {
			http.Error(w, "Could not shorten URL", http.StatusBadRequest)
			return
		}
		entry := t.URLEntry{
			UUID:        reqEntry.CorrelationID,
			ShortURL:    shortURL,
			OriginalURL: reqEntry.OriginalURL,
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
	respJSON, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}
