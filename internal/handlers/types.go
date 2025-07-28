package handlers

import (
	"github.com/repriest/url-shortener/internal/config"
	t "github.com/repriest/url-shortener/internal/storage/types"
)

type Handler struct {
	cfg *config.Config
	st  t.Storage
}

func NewHandler(cfg *config.Config, st t.Storage) *Handler {
	return &Handler{cfg: cfg, st: st}
}

type contextKey string

const userIDKey contextKey = "user_id"

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
	ShortURL      string `json:"short_url"`
}
