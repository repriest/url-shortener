package main

import (
	"bytes"
	"compress/gzip"
	"github.com/go-chi/chi/v5"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/handlers"
	"github.com/repriest/url-shortener/internal/storage/memory"
	"github.com/repriest/url-shortener/internal/zipper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var cfg *config.Config

func TestMain(m *testing.M) {
	var err error
	cfg, err = config.NewConfig()
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestShortenHandler(t *testing.T) {
	st, err := memory.NewMemoryStorage()
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)

	tt := []struct {
		name       string
		method     string
		body       string
		response   string
		statusCode int
	}{
		{
			name:       "EmptyBody",
			method:     http.MethodPost,
			body:       "",
			response:   "",
			statusCode: http.StatusCreated,
		},
		{
			name:       "ValidURL",
			method:     http.MethodPost,
			body:       "https://google.com",
			response:   cfg.BaseURL + "/aHR0cHM6Ly9nb29nbGUuY29t",
			statusCode: http.StatusCreated,
		},
		{
			name:       "InvalidURL",
			method:     http.MethodPost,
			body:       "badurl!@#$",
			response:   "Could not shorten URL\n",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, cfg.ServerAddr, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			h.ShortenHandler(rec, req)

			resp := rec.Result()
			respBody, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, tc.response, string(respBody))

		})
	}
}

func TestExpandHandler(t *testing.T) {
	st, err := memory.NewMemoryStorage()
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)
	r := chi.NewRouter()
	r.Get("/{id}", h.ExpandHandler)

	tt := []struct {
		name       string
		method     string
		path       string
		location   string
		statusCode int
	}{
		{
			name:       "ValidURL",
			method:     http.MethodGet,
			path:       "/aHR0cHM6Ly9nb29nbGUuY29t",
			location:   "https://google.com",
			statusCode: http.StatusTemporaryRedirect,
		},
		{
			name:       "InvalidURL",
			method:     http.MethodGet,
			path:       "/notbase64",
			location:   "",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)

			r.ServeHTTP(resp, req)

			assert.Equal(t, tc.statusCode, resp.Code)
			assert.Equal(t, tc.location, resp.Header().Get("Location"))
		})
	}
}

func TestShortenJSONHandler(t *testing.T) {
	st, err := memory.NewMemoryStorage()
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)

	tt := []struct {
		name        string
		method      string
		body        string
		response    string
		statusCode  int
		contentType string
	}{
		{
			name:        "ValidURL",
			method:      http.MethodPost,
			body:        `{"url":"https://google.com"}`,
			response:    `{"result":"` + cfg.BaseURL + `/aHR0cHM6Ly9nb29nbGUuY29t"}`,
			statusCode:  http.StatusCreated,
			contentType: "application/json",
		},
		{
			name:        "InvalidJSON",
			method:      http.MethodPost,
			body:        `{not: json}`,
			response:    "Invalid JSON\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "NoURL",
			method:      http.MethodPost,
			body:        `{"foo":"bar"}`,
			response:    "URL is required\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "InvalidURL",
			method:      http.MethodPost,
			body:        `{"url":"badurl!@#$"}`,
			response:    "Could not shorten URL\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/shorten", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			h.ShortenJSONHandler(rec, req)

			resp := rec.Result()
			respBody, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, tc.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.response, string(respBody))
		})
	}
}

func TestGzipCompression(t *testing.T) {
	st, err := memory.NewMemoryStorage()
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(zipper.GzipMiddleware)
		r.Post("/", h.ShortenHandler)
	})

	requestBody := "https://google.com"
	successBody := cfg.BaseURL + "/aHR0cHM6Ly9nb29nbGUuY29t"

	// compress test
	t.Run("sends_gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", buf)
		req.Header.Set("Content-Encoding", "gzip")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		resp := rec.Result()
		respBody, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, string(respBody), successBody)
	})

	// decompress test
	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/", buf)
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		resp := rec.Result()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		defer zr.Close()

		respBody, err := io.ReadAll(zr)
		require.NoError(t, err)

		require.Equal(t, string(respBody), successBody)
	})
}

func TestShortenBatchHandler(t *testing.T) {
	st, err := memory.NewMemoryStorage()
	require.NoError(t, err)
	defer st.Close()
	h := handlers.NewHandler(cfg, st)

	tt := []struct {
		name        string
		method      string
		body        string
		response    string
		statusCode  int
		contentType string
	}{
		{
			name:        "ValidBatch",
			method:      http.MethodPost,
			body:        `[{"correlation_id":"1","original_url":"https://google.com"}]`,
			response:    `[{"correlation_id":"1","short_url":"` + cfg.BaseURL + `/aHR0cHM6Ly9nb29nbGUuY29t"}]`,
			statusCode:  http.StatusCreated,
			contentType: "application/json",
		},
		{
			name:        "EmptyBatch",
			method:      http.MethodPost,
			body:        `[]`,
			response:    "Empty batch\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "InvalidJSON",
			method:      http.MethodPost,
			body:        `{not: json}`,
			response:    "Invalid JSON\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "EmptyURL",
			method:      http.MethodPost,
			body:        `[{"correlation_id":"1","original_url":""}]`,
			response:    "Empty URL\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
		{
			name:        "InvalidURL",
			method:      http.MethodPost,
			body:        `[{"correlation_id":"1","original_url":"badurl!@#$"}]`,
			response:    "Could not shorten URL\n",
			statusCode:  http.StatusBadRequest,
			contentType: "text/plain; charset=utf-8",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/shorten/batch", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			h.ShortenBatchHandler(rec, req)

			resp := rec.Result()
			respBody, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, tc.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.response, string(respBody))
		})
	}
}
