package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/repriest/url-shortener/internal/config"
	"github.com/repriest/url-shortener/internal/contextkeys"
	"net/http"
	"strings"
)

func signValue(value string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(value))
	signature := h.Sum(nil)
	encodedValue := base64.URLEncoding.EncodeToString([]byte(value))
	encodedSignature := base64.URLEncoding.EncodeToString(signature)
	return encodedValue + "." + encodedSignature
}

func verifyCookie(cookieValue string, secret string) (bool, string) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return false, ""
	}
	encodedValue := parts[0]
	encodedSignature := parts[1]
	valueBytes, err := base64.URLEncoding.DecodeString(encodedValue)
	if err != nil {
		return false, ""
	}
	signatureBytes, err := base64.URLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return false, ""
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(valueBytes)
	expectedSignature := h.Sum(nil)
	if subtle.ConstantTimeCompare(signatureBytes, expectedSignature) != 1 {
		return false, ""
	}
	return true, string(valueBytes)
}

func SetCookieMiddleware(cfg *config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			var userID string
			if err == nil && cookie != nil {
				isValid, id := verifyCookie(cookie.Value, cfg.CookieSecret)
				if isValid {
					userID = id
				}
			}
			if userID == "" {
				userID = uuid.New().String()
				signedValue := signValue(userID, cfg.CookieSecret)
				http.SetCookie(w, &http.Cookie{
					Name:     "session_id",
					Value:    signedValue,
					Path:     "/",
					HttpOnly: true,
					Secure:   false,
					SameSite: http.SameSiteStrictMode,
				})
			}
			ctx := context.WithValue(r.Context(), contextkeys.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthRequiredMiddleware(cfg *config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil || cookie == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			isValid, userID := verifyCookie(cookie.Value, cfg.CookieSecret)
			if !isValid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextkeys.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
