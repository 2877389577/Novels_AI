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
	// defaultLogOutput 是没有显式配置输出目标时的默认输出位置。
	defaultLogOutput = "console"
	// defaultLogLevel 是没有显式配置日志级别时的默认最低输出级别。
	defaultLogLevel = "info"
	// defaultLogFile 是文件输出模式下没有显式配置路径时使用的默认日志文件。
	defaultLogFile = "logs/novels_ai.log"
	// defaultLogMaxSizeMB 是单个日志文件的默认最大体积，单位 MB。
	defaultLogMaxSizeMB = 100
	// defaultLogMaxBackups 是默认保留的历史日志备份数量。
	defaultLogMaxBackups = 5
)

// NewLogger 根据配置创建 slog.Logger。
//
// 这里不直接设置全局默认 logger，调用方可以决定是否通过 slog.SetDefault 接管全局日志。
// 初始化过程分为两步：先解析日志级别，再根据 output 选择控制台 writer 或文件轮转 writer。
func NewLogger(config LogConfig) (*slog.Logger, error) {
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return nil, err
	}

	writer, err := newLogWriter(config)
	if err != nil {
		return nil, err
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler), nil
}

// newLogWriter 根据日志输出配置返回 slog handler 使用的 io.Writer。
//
// console/stdout 会直接写入标准输出；file 会创建带轮转能力的文件 writer。
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

// parseLogLevel 将配置文件中的字符串级别转换成 slog.Level。
//
// slog 的 handler 会丢弃低于该级别的日志，例如 level=warn 时 debug/info 都不会输出。
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

// rotatingFileWriter 是一个简单的按大小轮转的日志 writer。
//
// 它实现 io.Writer，因此可以直接交给 slog.NewTextHandler 使用。
// 该类型内部持有文件句柄和当前文件大小；每次写入前判断是否需要轮转。
type rotatingFileWriter struct {
	// mu 保护 file/size 的并发访问，避免多个 goroutine 同时写日志时互相覆盖状态。
	mu sync.Mutex
	// path 是当前活跃日志文件路径。
	path string
	// maxSize 是单个日志文件最大体积，单位字节。
	maxSize int64
	// maxBackups 是保留的备份文件数量，备份文件命名为 path.1、path.2 ...
	maxBackups int
	// file 是当前打开的日志文件句柄。
	file *os.File
	// size 记录当前活跃日志文件已写入的字节数，用于判断是否超过 maxSize。
	size int64
}

// newRotatingFileWriter 根据文件日志配置创建轮转 writer。
//
// 空配置会使用默认值；负数配置会被视为非法配置并返回错误，避免启动后出现不可预测的轮转行为。
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

// Write 实现 io.Writer。
//
// slog 每产生一条日志都会调用该方法；写入前先检查文件句柄，再按大小判断是否需要轮转。
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

// shouldRotate 判断本次写入是否会让当前日志文件超过 maxSize。
//
// size > 0 的条件可以避免空文件在第一条日志过大时先生成一个没有内容的备份文件。
func (w *rotatingFileWriter) shouldRotate(writeLen int) bool {
	return w.maxSize > 0 && w.size > 0 && w.size+int64(writeLen) > w.maxSize
}

// open 确保日志目录存在，并以追加模式打开当前活跃日志文件。
//
// 打开后会读取文件现有大小，这样程序重启后仍能基于已有文件大小继续轮转。
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

// rotate 执行一次完整轮转。
//
// 它会先关闭当前文件句柄，再移动历史备份文件，最后重新打开一个新的活跃日志文件。
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

// rotateBackups 维护备份文件序列。
//
// 当 maxBackups 为 0 时，不保留任何历史日志，只删除当前文件并重新开始写入。
// 当 maxBackups 大于 0 时，会先删除最旧备份，再从后往前重命名，避免覆盖较新的备份。
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

// renameLogFile 封装日志文件重命名。
//
// 源文件不存在时认为无需处理，因为首次轮转或备份数量不足时对应文件可能还没有生成。
func renameLogFile(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rename log file %q to %q: %w", oldPath, newPath, err)
	}
	return nil
}

// backupLogPath 根据备份序号生成备份文件名。
//
// 例如当前文件为 logs/novels_ai.log，则第 1 个备份为 logs/novels_ai.log.1。
func backupLogPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}
