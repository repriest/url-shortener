package baseurl

import (
	"encoding/base64"
	"net/url"
)

func ShortenURL(longURL string) (string, error) {
	if _, err := url.ParseRequestURI(longURL); err != nil {
		return "", err
	}
	shortURL := base64.StdEncoding.EncodeToString([]byte(longURL))
	return shortURL, nil
}

func ExpandURL(shortURL string) (string, error) {
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
