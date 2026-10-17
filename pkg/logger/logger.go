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
		if outputPath != "" {
			fileHandler := newFileHandler(outputPath, opts)
			handlers = append(handlers, fileHandler)
		} else {
			handlers = append(handlers, slog.NewJSONHandler(io.Discard, opts))
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
	currentDate string
	currentPart int
	file        *os.File
	mu          sync.Mutex
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	today := now.Format("2006-01-02")

	// Rotate when date changes
	if w.currentDate != "" && w.currentDate != today {
		if w.file != nil {
			_ = w.file.Close()
			w.file = nil
		}
		w.currentDate = today
		w.currentPart = 0
	} else if w.currentDate == "" {
		w.currentDate = today
	}

	// Rotate when file size exceeds maxSize
	if w.file != nil {
		if stat, err := w.file.Stat(); err == nil && stat.Size() >= w.maxSize {
			_ = w.file.Close()
			w.file = nil
			w.currentPart++
		}
	}

	if w.file == nil {
		filename := w.buildFilename(today, w.currentPart)
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return 0, err
		}
		w.file = f
	}

	return w.file.Write(p)
}

func (w *rotatingWriter) buildFilename(dateStr string, part int) string {
	dir := filepath.Dir(w.path)
	ext := filepath.Ext(w.path)
	base := strings.TrimSuffix(filepath.Base(w.path), ext)
	if part > 0 {
		return filepath.Join(dir, fmt.Sprintf("%s.%s.%d%s", base, dateStr, part, ext))
	}
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", base, dateStr, ext))
}

// GetLogDir returns the directory where log files are stored
func GetLogDir() string {
	if cfg.outputPath != "" {
		return filepath.Dir(cfg.outputPath)
	}
	return "logs"
}

// StartRetentionScheduler runs a background goroutine to clean up log files older than maxAge.
// It executes an initial scan on boot and repeats periodically based on interval.
func StartRetentionScheduler(ctx context.Context, dir string, maxAge time.Duration, interval time.Duration) {
	if dir == "" {
		dir = "logs"
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		cleanOldLogs(dir, maxAge)

		for {
			select {
			case <-ctx.Done():
				Info("log retention scheduler stopped")
				return
			case <-ticker.C:
				cleanOldLogs(dir, maxAge)
			}
		}
	}()
}

func cleanOldLogs(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	now := time.Now()
	deletedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(dir, entry.Name())
			if err := os.Remove(filePath); err == nil {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		Info("cleaned old log files", "deleted_count", deletedCount, "retention_days", int(maxAge.Hours()/24))
	}
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
