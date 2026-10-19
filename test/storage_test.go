package test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bubblevy/bookstore/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLocalStorageService(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bookstore-storage-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	svc := storage.NewLocalStorage(tempDir, "/api/v1/static")

	// Valid JPEG 1x1 image bytes
	validJPEG := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
		0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
		0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
		0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
		0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x1F, 0x00, 0x00,
		0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F,
		0x00, 0xBF, 0xFF, 0xD9,
	}

	opts := storage.UploadOptions{
		Folder:       "covers",
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
		MaxSizeBytes: 2 * 1024 * 1024,
	}

	t.Run("success upload valid JPEG", func(t *testing.T) {
		reader := bytes.NewReader(validJPEG)
		res, err := svc.Upload(context.Background(), reader, "cover.jpg", int64(len(validJPEG)), opts)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Contains(t, res.Key, "covers/")
		assert.Contains(t, res.URL, "/api/v1/static/covers/")
		assert.Equal(t, "image/jpeg", res.MimeType)
		assert.Equal(t, int64(len(validJPEG)), res.Size)

		// Verify file exists on disk
		savedFilePath := filepath.Join(tempDir, res.Key)
		_, err = os.Stat(savedFilePath)
		assert.NoError(t, err)

		// Test delete
		err = svc.Delete(context.Background(), res.URL)
		assert.NoError(t, err)
		_, err = os.Stat(savedFilePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("failed upload invalid MIME type", func(t *testing.T) {
		plainText := []byte("Hello world, this is a plain text file, not an image.")
		reader := bytes.NewReader(plainText)
		res, err := svc.Upload(context.Background(), reader, "test.txt", int64(len(plainText)), opts)
		assert.ErrorIs(t, err, storage.ErrInvalidFileType)
		assert.Nil(t, res)
	})

	t.Run("failed upload file exceeding max size", func(t *testing.T) {
		smallOpts := storage.UploadOptions{
			Folder:       "covers",
			AllowedTypes: []string{"image/jpeg"},
			MaxSizeBytes: 10, // 10 bytes limit
		}
		reader := bytes.NewReader(validJPEG)
		res, err := svc.Upload(context.Background(), reader, "cover.jpg", int64(len(validJPEG)), smallOpts)
		assert.ErrorIs(t, err, storage.ErrFileTooLarge)
		assert.Nil(t, res)
	})

	t.Run("failed upload empty file", func(t *testing.T) {
		reader := bytes.NewReader([]byte{})
		res, err := svc.Upload(context.Background(), reader, "empty.jpg", 0, opts)
		assert.ErrorIs(t, err, storage.ErrEmptyFile)
		assert.Nil(t, res)
	})

	t.Run("delete non-existing file returns ErrFileNotFound", func(t *testing.T) {
		err := svc.Delete(context.Background(), "/api/v1/static/covers/non-existent.jpg")
		assert.ErrorIs(t, err, storage.ErrFileNotFound)

		err = svc.Delete(context.Background(), "")
		assert.NoError(t, err)
	})

	t.Run("delete using full URL successfully deletes file", func(t *testing.T) {
		reader := bytes.NewReader(validJPEG)
		res, err := svc.Upload(context.Background(), reader, "full-url-test.jpg", int64(len(validJPEG)), opts)
		assert.NoError(t, err)

		fullURL := "http://localhost:8000" + res.URL
		err = svc.Delete(context.Background(), fullURL)
		assert.NoError(t, err)

		// Verify file is gone
		_, err = os.Stat(filepath.Join(tempDir, res.Key))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("get URL formatting", func(t *testing.T) {
		url := svc.GetURL("categories/fiction.png")
		assert.Equal(t, "/api/v1/static/categories/fiction.png", url)
	})

	t.Run("list files in folder", func(t *testing.T) {
		reader := bytes.NewReader(validJPEG)
		res, err := svc.Upload(context.Background(), reader, "list-test.jpg", int64(len(validJPEG)), opts)
		assert.NoError(t, err)

		files, err := svc.ListFiles(context.Background(), "covers")
		assert.NoError(t, err)
		assert.NotEmpty(t, files)

		var found bool
		for _, f := range files {
			if f.Key == res.Key {
				found = true
				assert.Equal(t, int64(len(validJPEG)), f.Size)
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("commit file moves uploads file to permanent location", func(t *testing.T) {
		tempOpts := storage.UploadOptions{
			Folder:       "uploads",
			AllowedTypes: []string{"image/jpeg"},
			MaxSizeBytes: 2 * 1024 * 1024,
		}
		reader := bytes.NewReader(validJPEG)
		res, err := svc.Upload(context.Background(), reader, "book-cover.jpg", int64(len(validJPEG)), tempOpts)
		assert.NoError(t, err)
		assert.Contains(t, res.Key, "uploads/")

		// Commit to permanent "covers"
		permKey, permURL, err := svc.CommitFile(context.Background(), res.URL, "covers")
		assert.NoError(t, err)
		assert.False(t, strings.HasPrefix(permKey, "uploads/"))
		assert.True(t, strings.HasPrefix(permKey, "covers/"))
		assert.Contains(t, permURL, "/covers/")

		// Verify new file exists on disk
		_, err = os.Stat(filepath.Join(tempDir, permKey))
		assert.NoError(t, err)

		// Verify old uploads file is gone
		_, err = os.Stat(filepath.Join(tempDir, res.Key))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("commit file moves temp file to permanent location", func(t *testing.T) {
		tempOpts := storage.UploadOptions{
			Folder:       "temp/covers",
			AllowedTypes: []string{"image/jpeg"},
			MaxSizeBytes: 2 * 1024 * 1024,
		}
		reader := bytes.NewReader(validJPEG)
		res, err := svc.Upload(context.Background(), reader, "book-cover.jpg", int64(len(validJPEG)), tempOpts)
		assert.NoError(t, err)
		assert.Contains(t, res.Key, "temp/covers/")

		// Commit to permanent "covers"
		permKey, permURL, err := svc.CommitFile(context.Background(), res.URL, "covers")
		assert.NoError(t, err)
		assert.False(t, strings.HasPrefix(permKey, "temp/"))
		assert.True(t, strings.HasPrefix(permKey, "covers/"))
		assert.Contains(t, permURL, "/covers/")

		// Verify new file exists on disk
		_, err = os.Stat(filepath.Join(tempDir, permKey))
		assert.NoError(t, err)

		// Verify old temp file is gone
		_, err = os.Stat(filepath.Join(tempDir, res.Key))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestStorageFactory(t *testing.T) {
	t.Run("factory default to local storage", func(t *testing.T) {
		cfg := storage.Config{
			Driver:         "local",
			LocalBaseDir:   "./public",
			LocalURLPrefix: "/api/v1/static",
		}
		svc, err := storage.NewStorageService(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("factory S3 driver requires bucket and region", func(t *testing.T) {
		cfg := storage.Config{
			Driver: "s3",
			S3:     storage.S3Config{},
		}
		svc, err := storage.NewStorageService(cfg)
		assert.Error(t, err)
		assert.Nil(t, svc)
	})

	t.Run("factory S3 driver success with bucket and region", func(t *testing.T) {
		cfg := storage.Config{
			Driver: "s3",
			S3: storage.S3Config{
				Bucket: "my-bookstore-bucket",
				Region: "ap-southeast-1",
			},
		}
		svc, err := storage.NewStorageService(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
		assert.Equal(t, "https://my-bookstore-bucket.s3.ap-southeast-1.amazonaws.com/covers/test.jpg", svc.GetURL("covers/test.jpg"))
	})
}

type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *MockS3Client) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.CopyObjectOutput), args.Error(1)
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func TestS3StorageService(t *testing.T) {
	validJPEG := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
	}

	cfg := storage.S3Config{
		Bucket: "test-bucket",
		Region: "ap-southeast-1",
	}

	t.Run("upload success calls PutObject on S3 client", func(t *testing.T) {
		mockClient := new(MockS3Client)
		svc := storage.NewS3StorageWithClient(cfg, mockClient)

		mockClient.On("PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
			return *input.Bucket == "test-bucket" &&
				strings.HasPrefix(*input.Key, "uploads/") &&
				*input.ContentType == "image/jpeg"
		})).Return(&s3.PutObjectOutput{}, nil).Once()

		opts := storage.UploadOptions{
			Folder:       "uploads",
			AllowedTypes: []string{"image/jpeg"},
			MaxSizeBytes: 2 * 1024 * 1024,
		}

		res, err := svc.Upload(context.Background(), bytes.NewReader(validJPEG), "cover.jpg", int64(len(validJPEG)), opts)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Contains(t, res.Key, "uploads/")
		assert.Equal(t, "image/jpeg", res.MimeType)
		mockClient.AssertExpectations(t)
	})

	t.Run("delete calls DeleteObject on S3 client", func(t *testing.T) {
		mockClient := new(MockS3Client)
		svc := storage.NewS3StorageWithClient(cfg, mockClient)

		mockClient.On("DeleteObject", mock.Anything, mock.MatchedBy(func(input *s3.DeleteObjectInput) bool {
			return *input.Bucket == "test-bucket" && *input.Key == "uploads/cover.jpg"
		})).Return(&s3.DeleteObjectOutput{}, nil).Once()

		err := svc.Delete(context.Background(), "https://test-bucket.s3.ap-southeast-1.amazonaws.com/uploads/cover.jpg")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("commit file copies object from uploads to target folder and deletes source", func(t *testing.T) {
		mockClient := new(MockS3Client)
		svc := storage.NewS3StorageWithClient(cfg, mockClient)

		sourceURL := "https://test-bucket.s3.ap-southeast-1.amazonaws.com/uploads/123456-abc.jpg"

		mockClient.On("CopyObject", mock.Anything, mock.MatchedBy(func(input *s3.CopyObjectInput) bool {
			return *input.Bucket == "test-bucket" &&
				*input.CopySource == "test-bucket/uploads/123456-abc.jpg" &&
				*input.Key == "covers/123456-abc.jpg"
		})).Return(&s3.CopyObjectOutput{}, nil).Once()

		mockClient.On("DeleteObject", mock.Anything, mock.MatchedBy(func(input *s3.DeleteObjectInput) bool {
			return *input.Bucket == "test-bucket" && *input.Key == "uploads/123456-abc.jpg"
		})).Return(&s3.DeleteObjectOutput{}, nil).Once()

		permKey, permURL, err := svc.CommitFile(context.Background(), sourceURL, "covers")
		assert.NoError(t, err)
		assert.Equal(t, "covers/123456-abc.jpg", permKey)
		assert.Equal(t, "https://test-bucket.s3.ap-southeast-1.amazonaws.com/covers/123456-abc.jpg", permURL)
		mockClient.AssertExpectations(t)
	})

	t.Run("commit file already in target folder does not call S3 copy", func(t *testing.T) {
		mockClient := new(MockS3Client)
		svc := storage.NewS3StorageWithClient(cfg, mockClient)

		alreadyPermURL := "https://test-bucket.s3.ap-southeast-1.amazonaws.com/covers/123456-abc.jpg"
		permKey, permURL, err := svc.CommitFile(context.Background(), alreadyPermURL, "covers")
		assert.NoError(t, err)
		assert.Equal(t, "covers/123456-abc.jpg", permKey)
		assert.Equal(t, alreadyPermURL, permURL)
		mockClient.AssertNotCalled(t, "CopyObject")
	})

	t.Run("list files returns stored file info list", func(t *testing.T) {
		mockClient := new(MockS3Client)
		svc := storage.NewS3StorageWithClient(cfg, mockClient)

		now := time.Now()
		mockClient.On("ListObjectsV2", mock.Anything, mock.MatchedBy(func(input *s3.ListObjectsV2Input) bool {
			return *input.Bucket == "test-bucket" && *input.Prefix == "uploads/"
		})).Return(&s3.ListObjectsV2Output{
			Contents: []s3types.Object{
				{
					Key:          aws.String("uploads/file-1.jpg"),
					LastModified: &now,
					Size:         aws.Int64(1024),
				},
			},
		}, nil).Once()

		files, err := svc.ListFiles(context.Background(), "uploads")
		assert.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "uploads/file-1.jpg", files[0].Key)
		assert.Equal(t, int64(1024), files[0].Size)
		mockClient.AssertExpectations(t)
	})

	t.Run("get URL supports CDN and custom endpoint", func(t *testing.T) {
		cdnCfg := storage.S3Config{
			Bucket:       "my-bucket",
			Region:       "ap-southeast-1",
			PublicCDNURL: "https://cdn.bookstore.com",
		}
		cdnSvc := storage.NewS3StorageWithClient(cdnCfg, nil)
		assert.Equal(t, "https://cdn.bookstore.com/covers/pic.jpg", cdnSvc.GetURL("covers/pic.jpg"))

		endpointCfg := storage.S3Config{
			Bucket:       "my-bucket",
			Region:       "us-east-1",
			Endpoint:     "http://localhost:9000",
			UsePathStyle: true,
		}
		endpointSvc := storage.NewS3StorageWithClient(endpointCfg, nil)
		assert.Equal(t, "http://localhost:9000/my-bucket/covers/pic.jpg", endpointSvc.GetURL("covers/pic.jpg"))
	})
}
