package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	RequestTimeout time.Duration
	AttachmentDir  string
	MaxUploadBytes int64
	AllowedOrigins []string
	WebDistDir     string
}

func Load() (Config, error) {
	timeout, err := time.ParseDuration(value("REQUEST_TIMEOUT", "5s"))
	if err != nil || timeout <= 0 {
		return Config{}, errors.New("REQUEST_TIMEOUT must be a positive duration")
	}
	maxBytes, err := strconv.ParseInt(value("MAX_UPLOAD_BYTES", "5242880"), 10, 64)
	if err != nil || maxBytes <= 0 {
		return Config{}, errors.New("MAX_UPLOAD_BYTES must be positive")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return Config{
		HTTPAddr:       value("HTTP_ADDR", ":8080"),
		DatabaseURL:    databaseURL,
		RequestTimeout: timeout,
		AttachmentDir:  value("ATTACHMENT_DIR", "./data/attachments"),
		MaxUploadBytes: maxBytes,
		AllowedOrigins: splitCSV(value("ALLOWED_ORIGINS", "http://localhost:5173")),
		WebDistDir:     value("WEB_DIST_DIR", "./web/dist"),
	}, nil
}

func value(key, fallback string) string {
	if current := strings.TrimSpace(os.Getenv(key)); current != "" {
		return current
	}
	return fallback
}

func splitCSV(raw string) []string {
	values := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}
