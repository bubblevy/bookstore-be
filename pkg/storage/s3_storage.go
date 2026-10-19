package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// S3Config holds connection and bucket details for AWS S3 and compatible services (MinIO, R2, Wasabi).
type S3Config struct {
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // Optional: custom endpoint for MinIO, Cloudflare R2, LocalStack
	PublicCDNURL    string // Optional: CloudFront or custom domain URL
	UsePathStyle    bool   // Optional: true for MinIO/LocalStack
}

// S3ClientAPI defines the subset of AWS S3 client methods utilized by S3StorageService.
type S3ClientAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type s3StorageService struct {
	cfg    S3Config
	client S3ClientAPI
}

// NewS3Storage initializes an S3-compatible cloud storage adapter using AWS SDK v2.
func NewS3Storage(cfg S3Config) (StorageService, error) {
	if cfg.Bucket == "" || cfg.Region == "" {
		return nil, errors.New("s3 storage requires S3_BUCKET and S3_REGION configuration")
	}

	var optFns []func(*awsconfig.LoadOptions) error
	optFns = append(optFns, awsconfig.WithRegion(cfg.Region))

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed loading AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return NewS3StorageWithClient(cfg, client), nil
}

// NewS3StorageWithClient creates a StorageService with a custom or mocked S3 client.
func NewS3StorageWithClient(cfg S3Config, client S3ClientAPI) StorageService {
	return &s3StorageService{
		cfg:    cfg,
		client: client,
	}
}

func (s *s3StorageService) Upload(ctx context.Context, file io.Reader, originalFilename string, size int64, opts UploadOptions) (*UploadResult, error) {
	if file == nil {
		return nil, ErrEmptyFile
	}
	if opts.MaxSizeBytes > 0 && size > opts.MaxSizeBytes {
		return nil, ErrFileTooLarge
	}

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

	// Reconstruct the full reader
	fullReader := io.MultiReader(bytes.NewReader(buf[:n]), file)

	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		switch detectedMime {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		default:
			ext = ".bin"
		}
	}

	randomPart := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	uniqueFilename := fmt.Sprintf("%d-%s%s", time.Now().Unix(), randomPart, ext)
	cleanFolder := strings.Trim(filepath.ToSlash(opts.Folder), "/")
	if cleanFolder == "" {
		cleanFolder = "uploads"
	}
	key := path.Join(cleanFolder, uniqueFilename)

	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.Bucket),
		Key:         aws.String(key),
		Body:        fullReader,
		ContentType: aws.String(detectedMime),
	}
	if size > 0 {
		putInput.ContentLength = aws.Int64(size)
	}

	if _, err := s.client.PutObject(ctx, putInput); err != nil {
		return nil, fmt.Errorf("%w: failed uploading to S3: %s", ErrUploadFailed, err)
	}

	publicURL := s.GetURL(key)

	return &UploadResult{
		Key:          key,
		URL:          publicURL,
		Filename:     uniqueFilename,
		OriginalName: originalFilename,
		Size:         size,
		MimeType:     detectedMime,
	}, nil
}

func (s *s3StorageService) Delete(ctx context.Context, fileKeyOrURL string) error {
	key := s.extractKey(fileKeyOrURL)
	if key == "" {
		return nil
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed deleting object from S3: %w", err)
	}

	return nil
}

func (s *s3StorageService) GetURL(fileKey string) string {
	cleanKey := strings.TrimLeft(filepath.ToSlash(fileKey), "/")
	if s.cfg.PublicCDNURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.PublicCDNURL, "/"), cleanKey)
	}
	if s.cfg.Endpoint != "" {
		if s.cfg.UsePathStyle {
			return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.cfg.Endpoint, "/"), s.cfg.Bucket, cleanKey)
		}
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.Endpoint, "/"), cleanKey)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.cfg.Bucket, s.cfg.Region, cleanKey)
}

func (s *s3StorageService) ListFiles(ctx context.Context, folder string) ([]StoredFileInfo, error) {
	cleanFolder := strings.Trim(filepath.ToSlash(folder), "/")
	prefix := cleanFolder
	if prefix != "" {
		prefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.cfg.Bucket),
		Prefix: aws.String(prefix),
	}

	var results []StoredFileInfo
	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed listing objects from S3: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			if *obj.Key == prefix {
				continue
			}
			var modTime time.Time
			if obj.LastModified != nil {
				modTime = *obj.LastModified
			}
			var size int64
			if obj.Size != nil {
				size = *obj.Size
			}
			results = append(results, StoredFileInfo{
				Key:     *obj.Key,
				ModTime: modTime,
				Size:    size,
			})
		}
	}

	return results, nil
}

func (s *s3StorageService) CommitFile(ctx context.Context, fileKeyOrURL string, targetFolder string) (string, string, error) {
	cleanKey := s.extractKey(fileKeyOrURL)
	if cleanKey == "" {
		return "", "", ErrFileNotFound
	}

	cleanTargetFolder := strings.Trim(filepath.ToSlash(targetFolder), "/")
	if cleanTargetFolder == "" {
		cleanTargetFolder = "covers"
	}

	// If already in target folder, no need to move
	if strings.HasPrefix(cleanKey, cleanTargetFolder+"/") {
		return cleanKey, s.GetURL(cleanKey), nil
	}

	// If not in temp/ or uploads/, it's already permanent
	if !strings.HasPrefix(cleanKey, "temp/") && !strings.HasPrefix(cleanKey, "uploads/") {
		return cleanKey, s.GetURL(cleanKey), nil
	}

	filename := path.Base(cleanKey)
	permanentKey := path.Join(cleanTargetFolder, filename)

	copySource := fmt.Sprintf("%s/%s", s.cfg.Bucket, cleanKey)
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.cfg.Bucket),
		Key:        aws.String(permanentKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			return "", "", ErrFileNotFound
		}
		return "", "", fmt.Errorf("failed copying file in S3: %w", err)
	}

	// Remove uncommitted source file from S3
	_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(cleanKey),
	})

	return permanentKey, s.GetURL(permanentKey), nil
}

func (s *s3StorageService) extractKey(fileKeyOrURL string) string {
	trimmed := strings.TrimSpace(fileKeyOrURL)
	if trimmed == "" {
		return ""
	}

	key := trimmed
	if s.cfg.PublicCDNURL != "" && strings.HasPrefix(key, s.cfg.PublicCDNURL) {
		key = strings.TrimPrefix(key, s.cfg.PublicCDNURL)
	} else if s.cfg.Endpoint != "" && strings.HasPrefix(key, s.cfg.Endpoint) {
		key = strings.TrimPrefix(key, s.cfg.Endpoint)
		key = strings.TrimPrefix(key, "/")
		if strings.HasPrefix(key, s.cfg.Bucket+"/") {
			key = strings.TrimPrefix(key, s.cfg.Bucket+"/")
		}
	} else if strings.Contains(key, fmt.Sprintf("%s.s3.", s.cfg.Bucket)) {
		parts := strings.Split(key, ".amazonaws.com/")
		if len(parts) == 2 {
			key = parts[1]
		}
	} else if u, err := url.Parse(trimmed); err == nil && u.Path != "" {
		key = u.Path
		trimmedPath := strings.TrimPrefix(key, "/")
		if strings.HasPrefix(trimmedPath, s.cfg.Bucket+"/") {
			key = strings.TrimPrefix(trimmedPath, s.cfg.Bucket+"/")
		}
	}

	cleanKey := strings.TrimLeft(filepath.ToSlash(key), "/")
	if idx := strings.Index(cleanKey, "api/v1/static/"); idx != -1 {
		cleanKey = cleanKey[idx+len("api/v1/static/"):]
	}

	return cleanKey
}
