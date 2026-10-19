package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/storage"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUploadUsecase struct {
	mock.Mock
}

func (m *MockUploadUsecase) UploadImage(ctx context.Context, file io.Reader, filename string, size int64, folder string) (*dto.UploadFileResponse, error) {
	args := m.Called(ctx, file, filename, size, folder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UploadFileResponse), args.Error(1)
}

func (m *MockUploadUsecase) DeleteFile(ctx context.Context, fileKeyOrURL string) error {
	args := m.Called(ctx, fileKeyOrURL)
	return args.Error(0)
}

func (m *MockUploadUsecase) CleanOrphanedFiles(ctx context.Context, olderThan time.Duration) (int, error) {
	args := m.Called(ctx, olderThan)
	return args.Int(0), args.Error(1)
}

func (m *MockUploadUsecase) StartOrphanedFilesCleanup(ctx context.Context, interval time.Duration, olderThan time.Duration) {
	m.Called(ctx, interval, olderThan)
}

type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) Upload(ctx context.Context, file io.Reader, originalFilename string, size int64, opts storage.UploadOptions) (*storage.UploadResult, error) {
	args := m.Called(ctx, file, originalFilename, size, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *MockStorageService) Delete(ctx context.Context, fileKeyOrURL string) error {
	args := m.Called(ctx, fileKeyOrURL)
	return args.Error(0)
}

func (m *MockStorageService) GetURL(fileKey string) string {
	args := m.Called(fileKey)
	return args.String(0)
}

func (m *MockStorageService) ListFiles(ctx context.Context, folder string) ([]storage.StoredFileInfo, error) {
	args := m.Called(ctx, folder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.StoredFileInfo), args.Error(1)
}

func (m *MockStorageService) CommitFile(ctx context.Context, fileKeyOrURL string, targetFolder string) (string, string, error) {
	args := m.Called(ctx, fileKeyOrURL, targetFolder)
	return args.String(0), args.String(1), args.Error(2)
}

func setupUploadApp(mockUC *MockUploadUsecase, allowAuth bool, allowAdmin bool) *fiber.App {
	app := fiber.New()

	dummyAuthMiddleware := func(c fiber.Ctx) error {
		if !allowAuth {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    401,
				"status":  "error",
				"message": "unauthorized",
			})
		}
		c.Locals("user_id", uint(1))
		return c.Next()
	}

	dummyAdminMiddleware := func(c fiber.Ctx) error {
		if !allowAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    403,
				"status":  "error",
				"message": "forbidden: admin role required",
			})
		}
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewUploadHandler(api, mockUC, dummyAuthMiddleware, dummyAdminMiddleware)

	return app
}

func TestUploadHandler(t *testing.T) {
	t.Run("success upload image returns 201 Created", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, true)

		mockUC.On("UploadImage", mock.Anything, mock.Anything, "cover.jpg", mock.Anything, "covers").
			Return(&dto.UploadFileResponse{
				FileKey:      "covers/1729255678-abc.jpg",
				FileURL:      "/api/v1/static/covers/1729255678-abc.jpg",
				FileName:     "1729255678-abc.jpg",
				OriginalName: "cover.jpg",
				FileSize:     1024,
				ContentType:  "image/jpeg",
			}, nil).Once()

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "cover.jpg")
		assert.NoError(t, err)
		_, _ = part.Write([]byte("dummy image content"))
		_ = writer.WriteField("folder", "covers")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var resBody struct {
			Code    int                    `json:"code"`
			Status  bool                   `json:"status"`
			Message string                 `json:"message"`
			Data    dto.UploadFileResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&resBody)
		assert.NoError(t, err)
		assert.True(t, resBody.Status)
		assert.Equal(t, "/api/v1/static/covers/1729255678-abc.jpg", resBody.Data.FileURL)
		mockUC.AssertExpectations(t)
	})

	t.Run("missing file returns 400 Bad Request", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, true)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("folder", "covers")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("unauthenticated request returns 401 Unauthorized", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, false, false)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "cover.jpg")
		_, _ = part.Write([]byte("image"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("non-admin request returns 403 Forbidden", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, false)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "cover.jpg")
		_, _ = part.Write([]byte("image"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("success delete file via query parameter returns 200 OK", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, true)

		mockUC.On("DeleteFile", mock.Anything, "/api/v1/static/covers/1729255678-abc.jpg").
			Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/uploads?file_url=/api/v1/static/covers/1729255678-abc.jpg", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resBody struct {
			Code    int    `json:"code"`
			Status  bool   `json:"status"`
			Message string `json:"message"`
		}
		err = json.NewDecoder(resp.Body).Decode(&resBody)
		assert.NoError(t, err)
		assert.True(t, resBody.Status)
		assert.Equal(t, "file deleted successfully", resBody.Message)
		mockUC.AssertExpectations(t)
	})

	t.Run("success delete file via file_key query parameter returns 200 OK", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, true)

		mockUC.On("DeleteFile", mock.Anything, "covers/1729255678-abc.jpg").
			Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/uploads?file_key=covers/1729255678-abc.jpg", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("delete missing target returns 400 Bad Request", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, true)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/uploads", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("delete failure returns 500 Internal Server Error", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, true)

		mockUC.On("DeleteFile", mock.Anything, "covers/error.jpg").
			Return(errors.New("disk failure")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/uploads?file_key=covers/error.jpg", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("delete file not found returns 404 Not Found", func(t *testing.T) {
		mockUC := new(MockUploadUsecase)
		app := setupUploadApp(mockUC, true, true)

		mockUC.On("DeleteFile", mock.Anything, "covers/nonexistent.jpg").
			Return(storage.ErrFileNotFound).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/uploads?file_key=covers/nonexistent.jpg", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestUploadUsecase(t *testing.T) {
	t.Run("success upload image delegates to storage service with uploads folder and 2MB limit", func(t *testing.T) {
		mockStorage := new(MockStorageService)
		uc := usecase.NewUploadUsecase(mockStorage)

		fileReader := bytes.NewReader([]byte("dummy content"))
		mockStorage.On("Upload", mock.Anything, fileReader, "category.png", int64(13), mock.MatchedBy(func(opts storage.UploadOptions) bool {
			return opts.Folder == "uploads" && opts.MaxSizeBytes == storage.MaxUploadSizeBytes
		})).Return(&storage.UploadResult{
			Key:          "uploads/cat-1.png",
			URL:          "http://localhost:8000/uploads/cat-1.png",
			Filename:     "cat-1.png",
			OriginalName: "category.png",
			Size:         13,
			MimeType:     "image/png",
		}, nil).Once()

		res, err := uc.UploadImage(context.Background(), fileReader, "category.png", 13, "")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "http://localhost:8000/uploads/cat-1.png", res.FileURL)
		mockStorage.AssertExpectations(t)
	})

	t.Run("delete file delegates to storage service", func(t *testing.T) {
		mockStorage := new(MockStorageService)
		uc := usecase.NewUploadUsecase(mockStorage)

		mockStorage.On("Delete", mock.Anything, "uploads/test.jpg").Return(nil).Once()

		err := uc.DeleteFile(context.Background(), "uploads/test.jpg")
		assert.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("clean orphaned files deletes abandoned temp uploads older than threshold", func(t *testing.T) {
		mockStorage := new(MockStorageService)
		uc := usecase.NewUploadUsecase(mockStorage)

		oldTime := time.Now().Add(-3 * time.Hour)
		mockStorage.On("ListFiles", mock.Anything, "uploads").Return([]storage.StoredFileInfo{
			{Key: "uploads/old-abandoned.jpg", ModTime: oldTime, Size: 1024},
		}, nil).Once()
		mockStorage.On("ListFiles", mock.Anything, "temp/covers").Return([]storage.StoredFileInfo{}, nil).Once()
		mockStorage.On("ListFiles", mock.Anything, "temp/categories").Return([]storage.StoredFileInfo{}, nil).Once()
		mockStorage.On("ListFiles", mock.Anything, "temp/uploads").Return([]storage.StoredFileInfo{}, nil).Once()

		mockStorage.On("Delete", mock.Anything, "uploads/old-abandoned.jpg").Return(nil).Once()

		deleted, err := uc.CleanOrphanedFiles(context.Background(), 2*time.Hour)
		assert.NoError(t, err)
		assert.Equal(t, 1, deleted)
		mockStorage.AssertExpectations(t)
	})
}
