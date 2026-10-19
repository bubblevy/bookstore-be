package usecase

import (
	"context"
	"io"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/bubblevy/bookstore/pkg/storage"
	"gorm.io/gorm"
)

// UploadUsecase manages file upload operations, validation, and orphan file garbage collection.
type UploadUsecase interface {
	UploadImage(ctx context.Context, file io.Reader, filename string, size int64, folder string) (*dto.UploadFileResponse, error)
	DeleteFile(ctx context.Context, fileKeyOrURL string) error
	CleanOrphanedFiles(ctx context.Context, olderThan time.Duration) (int, error)
	StartOrphanedFilesCleanup(ctx context.Context, interval time.Duration, olderThan time.Duration)
}

type uploadUsecase struct {
	storage storage.StorageService
	db      *gorm.DB
}

func NewUploadUsecase(storage storage.StorageService, db ...*gorm.DB) UploadUsecase {
	var gdb *gorm.DB
	if len(db) > 0 {
		gdb = db[0]
	}
	return &uploadUsecase{
		storage: storage,
		db:      gdb,
	}
}

func (u *uploadUsecase) UploadImage(ctx context.Context, file io.Reader, filename string, size int64, folder string) (*dto.UploadFileResponse, error) {
	opts := storage.UploadOptions{
		Folder:       "uploads",
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
		MaxSizeBytes: storage.MaxUploadSizeBytes, // 2MB limit
	}

	res, err := u.storage.Upload(ctx, file, filename, size, opts)
	if err != nil {
		return nil, err
	}

	return &dto.UploadFileResponse{
		FileKey:      res.Key,
		FileURL:      res.URL,
		FileName:     res.Filename,
		OriginalName: res.OriginalName,
		FileSize:     res.Size,
		ContentType:  res.MimeType,
	}, nil
}

func (u *uploadUsecase) DeleteFile(ctx context.Context, fileKeyOrURL string) error {
	return u.storage.Delete(ctx, fileKeyOrURL)
}

func (u *uploadUsecase) CleanOrphanedFiles(ctx context.Context, olderThan time.Duration) (int, error) {
	tempFolders := []string{"uploads", "temp/covers", "temp/categories", "temp/uploads"}
	deletedCount := 0

	for _, folder := range tempFolders {
		files, err := u.storage.ListFiles(ctx, folder)
		if err != nil {
			logger.Warn("failed listing files in temp folder for orphan cleanup", "folder", folder, "error", err)
			continue
		}

		for _, file := range files {
			// Skip files newer than the threshold to avoid deleting in-progress uploads
			if time.Since(file.ModTime) <= olderThan {
				continue
			}

			if err := u.storage.Delete(ctx, file.Key); err == nil {
				deletedCount++
				logger.Info("deleted abandoned temp upload", "key", file.Key, "age", time.Since(file.ModTime))
			}
		}
	}

	return deletedCount, nil
}

func (u *uploadUsecase) StartOrphanedFilesCleanup(ctx context.Context, interval time.Duration, olderThan time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("orphaned files cleanup stopped")
				return
			case <-ticker.C:
				deleted, err := u.CleanOrphanedFiles(ctx, olderThan)
				if err != nil {
					logger.Error("orphaned files cleanup error", "error", err)
				} else if deleted > 0 {
					logger.Info("orphaned files cleanup completed", "deleted", deleted)
				}
			}
		}
	}()
}
