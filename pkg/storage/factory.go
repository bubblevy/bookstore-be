package storage

import (
	"strings"
)

// Config holds configuration parameters for the storage provider.
type Config struct {
	Driver         string // "local" or "s3"
	LocalBaseDir   string // Directory for local storage, e.g. "./public"
	LocalURLPrefix string // URL prefix for serving static files, e.g. "/api/v1/static"
	S3             S3Config
}

// NewStorageService is the factory function that creates a StorageService instance based on the configured driver.
func NewStorageService(cfg Config) (StorageService, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "s3":
		return NewS3Storage(cfg.S3)
	default:
		return NewLocalStorage(cfg.LocalBaseDir, cfg.LocalURLPrefix), nil
	}
}
