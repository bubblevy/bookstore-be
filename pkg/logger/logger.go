package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request_id"

type config struct {
	env        string
	outputPath string
	level      slog.Level
}

var (
	log     *slog.Logger
	cfg     config
	writers []io.Writer
)

func Init(env, outputPath string) {
	cfg = config{
		env:        strings.ToLower(env),
		outputPath: outputPath,
		level:      slog.LevelInfo,
	}

	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     cfg.level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.LevelKey:
				a.Key = "level"
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	}

	var handlers []slog.Handler

	if cfg.env == "production" || cfg.env == "prod" {
		handlers = append(handlers, slog.NewJSONHandler(os.Stdout, opts))
		if outputPath != "" {
			fileHandler := newFileHandler(outputPath, opts)
			handlers = append(handlers, fileHandler)
		}
	} else {
		handlers = append(handlers, slog.NewTextHandler(os.Stdout, opts))
	}

	if len(handlers) == 1 {
		log = slog.New(handlers[0])
	} else {
		log = slog.New(slog.NewMultiHandler(handlers...))
	}

	slog.SetDefault(log)
}

func newFileHandler(outputPath string, opts *slog.HandlerOptions) slog.Handler {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to create log directory: %v\n", err)
		return slog.NewJSONHandler(io.Discard, opts)
	}

	w := &rotatingWriter{
		path:    outputPath,
		maxSize: 100 * 1024 * 1024, // 100MB per file
	}

	writers = append(writers, w)

	return slog.NewJSONHandler(w, opts)
}

type rotatingWriter struct {
	path        string
	maxSize     int64
	currentPath string
	file        *os.File
	mu          sync.Mutex
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// check if rotation needed
	if w.file != nil {
		size, _ := w.file.Stat()
		if size.Size() >= w.maxSize {
			w.file.Close()
			w.file = nil
		}
	}

	filename := w.currentPath
	if filename == "" {
		filename = w.buildFilename()
		w.currentPath = filename
	}

	if w.file == nil {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return 0, err
		}
		w.file = f
	}

	return w.file.Write(p)
}

func (w *rotatingWriter) buildFilename() string {
	now := time.Now()
	dir := filepath.Dir(w.path)
	ext := filepath.Ext(w.path)
	base := strings.TrimSuffix(filepath.Base(w.path), ext)
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", base, now.Format("2006-01-02"), ext))
}

func Sync() {
	for _, w := range writers {
		if f, ok := w.(*rotatingWriter); ok {
			f.mu.Lock()
			if f.file != nil {
				f.file.Sync()
			}
			f.mu.Unlock()
		}
	}
}

// WithRequestID returns a logger with request_id attached
func WithRequestID(requestID string) *slog.Logger {
	return log.With("request_id", requestID)
}

// WithUserID returns a logger with user_id attached
func WithUserID(userID uint) *slog.Logger {
	return log.With("user_id", userID)
}

// WithComponent returns a logger with component attached
func WithComponent(component string) *slog.Logger {
	return log.With("component", component)
}

// WithField returns a logger with an arbitrary field
func WithField(key string, value any) *slog.Logger {
	return log.With(key, value)
}

// WithFields returns a logger with multiple arbitrary fields
func WithFields(fields map[string]any) *slog.Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return log.With(args...)
}

// Debug logs at debug level
func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

// Info logs at info level
func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	log.Error(msg, args...)
}

// Fatal logs at error level and exits
func Fatal(msg string, args ...any) {
	log.Error(msg, args...)
	os.Exit(1)
}

// WithContext returns a logger with request_id from context
func WithContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return log
	}
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return log.With("request_id", requestID)
	}
	return log
}

// SetRequestID adds request_id to context
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}
