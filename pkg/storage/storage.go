package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrInvalidFileType = errors.New("invalid file type: only JPEG, PNG, and WebP images are allowed")
	ErrFileTooLarge    = errors.New("file size exceeds maximum allowed limit (2MB)")
	ErrEmptyFile       = errors.New("uploaded file cannot be empty")
	ErrUploadFailed    = errors.New("failed to upload file")
	ErrFileNotFound    = errors.New("file not found in storage")
)

const (
	// MaxUploadSizeBytes defines the maximum allowed file size for uploads (2MB).
	MaxUploadSizeBytes int64 = 2 * 1024 * 1024
)

// UploadOptions provides configurable options for file uploads.
type UploadOptions struct {
	Folder       string   // Target folder, e.g., "covers", "categories"
	AllowedTypes []string // MIME types, e.g., []string{"image/jpeg", "image/png", "image/webp"}
	MaxSizeBytes int64    // Maximum allowed file size in bytes
}

// UploadResult contains the metadata of a successfully uploaded file.
type UploadResult struct {
	Key          string `json:"file_key"`      // Storage identifier, e.g. "covers/1729255678-uuid.jpg"
	URL          string `json:"file_url"`      // Public URL to access the file
	Filename     string `json:"file_name"`     // Generated unique filename
	OriginalName string `json:"original_name"`  // Original client filename
	Size         int64  `json:"file_size"`     // File size in bytes
	MimeType     string `json:"content_type"`  // Detected MIME type
}

// StoredFileInfo holds metadata of a file stored in the storage backend.
type StoredFileInfo struct {
	Key     string
	ModTime time.Time
	Size    int64
}

// StorageService defines the contract for object storage providers (Local disk, AWS S3, MinIO, GCS, Cloudflare R2).
type StorageService interface {
	Upload(ctx context.Context, file io.Reader, originalFilename string, size int64, opts UploadOptions) (*UploadResult, error)
	Delete(ctx context.Context, fileKeyOrURL string) error
	GetURL(fileKey string) string
	ListFiles(ctx context.Context, folder string) ([]StoredFileInfo, error)
	CommitFile(ctx context.Context, fileKeyOrURL string, targetFolder string) (permanentKey string, permanentURL string, err error)
}
