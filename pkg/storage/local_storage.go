package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type localStorageService struct {
	baseDir   string // Directory on local disk, e.g., "./public"
	urlPrefix string // URL prefix for serving static files, e.g., "http://localhost:8000/covers" or "/api/v1/static"
}

// NewLocalStorage initializes a local disk storage provider.
func NewLocalStorage(baseDir, urlPrefix string) StorageService {
	cleanBase := strings.TrimRight(filepath.Clean(baseDir), "/\\")
	if cleanBase == "" {
		cleanBase = "./public"
	}
	cleanPrefix := strings.TrimSpace(urlPrefix)
	if cleanPrefix != "" && !strings.HasPrefix(cleanPrefix, "http://") && !strings.HasPrefix(cleanPrefix, "https://") {
		cleanPrefix = "/" + strings.Trim(cleanPrefix, "/")
	}
	cleanPrefix = strings.TrimRight(cleanPrefix, "/")
	return &localStorageService{
		baseDir:   cleanBase,
		urlPrefix: cleanPrefix,
	}
}

func (s *localStorageService) Upload(ctx context.Context, file io.Reader, originalFilename string, size int64, opts UploadOptions) (*UploadResult, error) {
	if file == nil {
		return nil, ErrEmptyFile
	}
	if opts.MaxSizeBytes > 0 && size > opts.MaxSizeBytes {
		return nil, ErrFileTooLarge
	}

	// Read initial 512 bytes for MIME type sniffing
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("%w: failed reading file header: %s", ErrUploadFailed, err)
	}
	if n == 0 {
		return nil, ErrEmptyFile
	}

	detectedMime := http.DetectContentType(buf[:n])
	if idx := strings.Index(detectedMime, ";"); idx != -1 {
		detectedMime = strings.TrimSpace(detectedMime[:idx])
	}

	// Reconstruct the stream
	fullReader := io.MultiReader(bytes.NewReader(buf[:n]), file)

	// Validate allowed MIME types
	if len(opts.AllowedTypes) > 0 {
		allowed := false
		for _, t := range opts.AllowedTypes {
			if strings.EqualFold(t, detectedMime) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, ErrInvalidFileType
		}
	}

	// Sanitize folder
	cleanFolder := filepath.Clean(opts.Folder)
	if cleanFolder == "." || strings.HasPrefix(cleanFolder, "..") || cleanFolder == "/" || cleanFolder == "\\" {
		cleanFolder = "uploads"
	}

	destDir := filepath.Join(s.baseDir, cleanFolder)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: failed creating upload directory: %s", ErrUploadFailed, err)
	}

	// Determine file extension
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		switch detectedMime {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		default:
			ext = ".bin"
		}
	}

	// Generate a collision-free safe filename
	randomPart := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	uniqueFilename := fmt.Sprintf("%d-%s%s", time.Now().Unix(), randomPart, ext)
	destPath := filepath.Join(destDir, uniqueFilename)

	dstFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed creating file on disk: %s", ErrUploadFailed, err)
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, fullReader)
	if err != nil {
		_ = os.Remove(destPath)
		return nil, fmt.Errorf("%w: failed writing file to disk: %s", ErrUploadFailed, err)
	}

	if opts.MaxSizeBytes > 0 && written > opts.MaxSizeBytes {
		_ = os.Remove(destPath)
		return nil, ErrFileTooLarge
	}

	slashFolder := filepath.ToSlash(cleanFolder)
	key := path.Join(slashFolder, uniqueFilename)
	publicURL := fmt.Sprintf("%s/%s", strings.TrimRight(s.urlPrefix, "/"), key)

	return &UploadResult{
		Key:          key,
		URL:          publicURL,
		Filename:     uniqueFilename,
		OriginalName: originalFilename,
		Size:         written,
		MimeType:     detectedMime,
	}, nil
}

func (s *localStorageService) Delete(ctx context.Context, fileKeyOrURL string) error {
	trimmed := strings.TrimSpace(fileKeyOrURL)
	if trimmed == "" {
		return nil
	}

	key := trimmed
	// If it's a full URL (http:// or https://), parse the URL path
	if u, err := url.Parse(trimmed); err == nil && u.Path != "" {
		key = u.Path
	}

	cleanPath := filepath.ToSlash(key)

	// Strip urlPrefix path if urlPrefix was configured
	if s.urlPrefix != "" {
		prefixPath := s.urlPrefix
		if pu, err := url.Parse(s.urlPrefix); err == nil && pu.Path != "" {
			prefixPath = pu.Path
		}
		prefixPath = strings.TrimRight(prefixPath, "/")
		if prefixPath != "" && strings.HasPrefix(cleanPath, prefixPath) {
			cleanPath = strings.TrimPrefix(cleanPath, prefixPath)
		}
	}

	// Also strip common prefix /api/v1/static/
	if idx := strings.Index(cleanPath, "/api/v1/static/"); idx != -1 {
		cleanPath = cleanPath[idx+len("/api/v1/static/"):]
	}

	cleanKey := strings.TrimLeft(cleanPath, "/")

	// Protect against directory traversal
	cleanKey = filepath.ToSlash(filepath.Clean(cleanKey))
	if strings.HasPrefix(cleanKey, "..") || cleanKey == "." {
		return nil
	}

	targetPath := filepath.Join(s.baseDir, filepath.FromSlash(cleanKey))
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return ErrFileNotFound
	}
	if err := os.Remove(targetPath); err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	return nil
}

func (s *localStorageService) GetURL(fileKey string) string {
	cleanKey := strings.TrimLeft(filepath.ToSlash(fileKey), "/")
	if s.urlPrefix == "" {
		return "/" + cleanKey
	}
	return fmt.Sprintf("%s/%s", strings.TrimRight(s.urlPrefix, "/"), cleanKey)
}

func (s *localStorageService) ListFiles(ctx context.Context, folder string) ([]StoredFileInfo, error) {
	cleanFolder := filepath.Clean(folder)
	dirPath := filepath.Join(s.baseDir, cleanFolder)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []StoredFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		key := path.Join(filepath.ToSlash(cleanFolder), entry.Name())
		result = append(result, StoredFileInfo{
			Key:     key,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}
	return result, nil
}

func (s *localStorageService) CommitFile(ctx context.Context, fileKeyOrURL string, targetFolder string) (string, string, error) {
	trimmed := strings.TrimSpace(fileKeyOrURL)
	if trimmed == "" {
		return "", "", ErrFileNotFound
	}

	key := trimmed
	if u, err := url.Parse(trimmed); err == nil && u.Path != "" {
		key = u.Path
	}

	cleanPath := filepath.ToSlash(key)
	if s.urlPrefix != "" {
		prefixPath := s.urlPrefix
		if pu, err := url.Parse(s.urlPrefix); err == nil && pu.Path != "" {
			prefixPath = pu.Path
		}
		prefixPath = strings.TrimRight(prefixPath, "/")
		if prefixPath != "" && strings.HasPrefix(cleanPath, prefixPath) {
			cleanPath = strings.TrimPrefix(cleanPath, prefixPath)
		}
	}

	if idx := strings.Index(cleanPath, "/api/v1/static/"); idx != -1 {
		cleanPath = cleanPath[idx+len("/api/v1/static/"):]
	}

	cleanKey := strings.TrimLeft(cleanPath, "/")
	cleanKey = filepath.ToSlash(filepath.Clean(cleanKey))

	cleanTargetFolder := strings.Trim(filepath.ToSlash(targetFolder), "/")
	if cleanTargetFolder == "" {
		cleanTargetFolder = "covers"
	}

	// If already in target folder, no need to move
	if strings.HasPrefix(cleanKey, cleanTargetFolder+"/") {
		return cleanKey, s.GetURL(cleanKey), nil
	}

	// If already in permanent folder (not in temp/ or uploads/), it's already committed
	if !strings.HasPrefix(cleanKey, "temp/") && !strings.HasPrefix(cleanKey, "uploads/") {
		return cleanKey, s.GetURL(cleanKey), nil
	}

	srcPath := filepath.Join(s.baseDir, filepath.FromSlash(cleanKey))
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", "", ErrFileNotFound
	}

	filename := path.Base(cleanKey)
	permanentKey := path.Join(cleanTargetFolder, filename)
	dstPath := filepath.Join(s.baseDir, filepath.FromSlash(permanentKey))

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", "", fmt.Errorf("failed creating target directory: %w", err)
	}

	// Move file: atomic rename or fallback copy+delete
	if err := os.Rename(srcPath, dstPath); err != nil {
		input, rErr := os.ReadFile(srcPath)
		if rErr != nil {
			return "", "", fmt.Errorf("failed moving file from temp: %w", err)
		}
		if wErr := os.WriteFile(dstPath, input, 0644); wErr != nil {
			return "", "", fmt.Errorf("failed writing file to permanent location: %w", wErr)
		}
		_ = os.Remove(srcPath)
	}

	return permanentKey, s.GetURL(permanentKey), nil
}
