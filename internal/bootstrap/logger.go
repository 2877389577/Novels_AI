package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	defaultLogOutput     = "console"
	defaultLogLevel      = "info"
	defaultLogFile       = "logs/novels_ai.log"
	defaultLogMaxSizeMB  = 100
	defaultLogMaxBackups = 5
)

func NewLogger(config LogConfig) (*slog.Logger, error) {
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return nil, err
	}

	writer, err := newLogWriter(config)
	if err != nil {
		return nil, err
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	return slog.New(handler), nil
}

func newLogWriter(config LogConfig) (io.Writer, error) {
	switch strings.ToLower(strings.TrimSpace(config.Output)) {
	case "", defaultLogOutput, "stdout":
		return os.Stdout, nil
	case "file":
		return newRotatingFileWriter(config.File)
	default:
		return nil, fmt.Errorf("unsupported log output %q", config.Output)
	}
}

func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", defaultLogLevel:
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported log level %q", level)
	}
}

type rotatingFileWriter struct {
	mu         sync.Mutex
	path       string
	maxSize    int64
	maxBackups int
	file       *os.File
	size       int64
}

func newRotatingFileWriter(config LogFileConfig) (*rotatingFileWriter, error) {
	path := strings.TrimSpace(config.Path)
	if path == "" {
		path = defaultLogFile
	}

	maxSizeMB := config.MaxSizeMB
	if maxSizeMB == 0 {
		maxSizeMB = defaultLogMaxSizeMB
	}
	if maxSizeMB < 0 {
		return nil, fmt.Errorf("log max_size_mb must be greater than zero")
	}

	maxBackups := config.MaxBackups
	if maxBackups == 0 {
		maxBackups = defaultLogMaxBackups
	}
	if maxBackups < 0 {
		return nil, fmt.Errorf("log max_backups must be greater than or equal to zero")
	}

	writer := &rotatingFileWriter{
		path:       path,
		maxSize:    maxSizeMB * 1024 * 1024,
		maxBackups: maxBackups,
	}
	if err := writer.open(); err != nil {
		return nil, err
	}

	return writer, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}

	if w.shouldRotate(len(p)) {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingFileWriter) shouldRotate(writeLen int) bool {
	return w.maxSize > 0 && w.size > 0 && w.size+int64(writeLen) > w.maxSize
}

func (w *rotatingFileWriter) open() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file %q: %w", w.path, err)
	}

	info, err := file.Stat()
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return fmt.Errorf("stat log file %q: %w; close log file: %w", w.path, err, closeErr)
		}
		return fmt.Errorf("stat log file %q: %w", w.path, err)
	}

	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rotatingFileWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close log file before rotate: %w", err)
		}
		w.file = nil
		w.size = 0
	}

	if err := w.rotateBackups(); err != nil {
		return err
	}

	return w.open()
}

func (w *rotatingFileWriter) rotateBackups() error {
	if w.maxBackups == 0 {
		if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove log file %q: %w", w.path, err)
		}
		return nil
	}

	oldest := backupLogPath(w.path, w.maxBackups)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove oldest log backup %q: %w", oldest, err)
	}

	for i := w.maxBackups - 1; i >= 1; i-- {
		oldPath := backupLogPath(w.path, i)
		newPath := backupLogPath(w.path, i+1)
		if err := renameLogFile(oldPath, newPath); err != nil {
			return err
		}
	}

	return renameLogFile(w.path, backupLogPath(w.path, 1))
}

func renameLogFile(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rename log file %q to %q: %w", oldPath, newPath, err)
	}
	return nil
}

func backupLogPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}
