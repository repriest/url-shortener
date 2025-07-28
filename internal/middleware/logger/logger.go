package logger

import (
	"bytes"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
		body   bytes.Buffer
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

// Log будет доступен всему коду как синглтон.
// Никакой код навыка, кроме функции Initialize, не должен модифицировать эту переменную.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровеньs
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}

// RequestLogger — middleware-логер для входящих HTTP-запросов.
func RequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		Log.Info("incoming HTTP request",
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.String("content_type", r.Header.Get("Content-Type")),
			zap.ByteString("body", bodyBytes),
		)

		h.ServeHTTP(w, r)

		Log.Info("request completed",
			zap.Duration("duration", time.Since(start)),
			zap.String("content_type", r.Header.Get("Content-Type")),
			zap.String("location", r.Header.Get("Location")),
		)
	})
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// сохраняем тело ответа для логирования
	r.responseData.body.Write(b)
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

func ResponseLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseData := &responseData{
			status: http.StatusOK, // по умолчанию 200 OK
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
			responseData:   responseData,
		}

		h.ServeHTTP(&lw, r)
		var responseBody string
		if responseData.body.Len() > 0 {
			responseBody = responseData.body.String()
		}

		Log.Info("outgoing HTTP response",
			zap.Int("status", responseData.status),
			zap.String("request_URI", r.URL.String()),
			zap.String("response_body", responseBody),
			zap.String("content_type", w.Header().Get("Content-Type")),
		)
	})
}
