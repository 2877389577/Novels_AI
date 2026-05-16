package bootstrap

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server" yaml:"server"`
	System SystemConfig `mapstructure:"system" yaml:"system"`
	// RateLimit 保存全局 HTTP 请求限速配置。
	RateLimit RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
	// Session 保存浏览器会话相关配置。
	Session SessionConfig `mapstructure:"session" yaml:"session"`
	// Log 保存日志服务相关配置，用于控制日志输出位置、日志级别以及文件轮转策略。
	Log LogConfig `mapstructure:"log" yaml:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port" yaml:"port"`
	Host string `mapstructure:"host" yaml:"host"`
}

// RateLimitConfig 定义全局 HTTP 请求限速配置。
type RateLimitConfig struct {
	// RequestsPerMinute 表示每分钟允许通过的请求数量；小于等于 0 时关闭限速。
	RequestsPerMinute int `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
}

// SessionConfig 定义浏览器会话配置。
type SessionConfig struct {
	// Salt 用于 gin session cookie store 的签名/加密盐。
	Salt string `mapstructure:"salt" yaml:"salt"`
}

// LogConfig 定义日志服务的顶层配置。
//
// Output 决定日志输出到控制台还是文件；Level 决定 slog 处理器允许输出的最低日志级别；
// File 仅在 Output 为 file 时生效，用于描述日志文件路径和轮转参数。
type LogConfig struct {
	// Output 支持 console/stdout/file；为空时使用默认值 console。
	Output string `mapstructure:"output" yaml:"output"`
	// Level 支持 debug/info/warn/error；为空时使用默认值 info。
	Level string `mapstructure:"level" yaml:"level"`
	// File 是文件输出模式下的详细配置。
	File LogFileConfig `mapstructure:"file" yaml:"file"`
}

// LogFileConfig 定义日志写入文件时的轮转配置。
//
// 当前实现按文件大小轮转：当现有日志文件加上本次写入内容超过 MaxSizeMB 时，
// 当前文件会被移动为 .1，历史备份依次后移，超过 MaxBackups 的最旧备份会被删除。
type LogFileConfig struct {
	// Path 是当前活跃日志文件路径；为空时使用 logs/novels_ai.log。
	Path string `mapstructure:"path" yaml:"path"`
	// MaxSizeMB 是单个日志文件的最大体积，单位 MB；为空或 0 时使用默认值 100。
	MaxSizeMB int64 `mapstructure:"max_size_mb" yaml:"max_size_mb"`
	// MaxBackups 是保留的历史备份文件数量；为空或 0 时使用默认值 5。
	MaxBackups int `mapstructure:"max_backups" yaml:"max_backups"`
}

type SystemConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres" yaml:"postgres"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host" yaml:"host"`
	Port     int    `mapstructure:"port" yaml:"port"`
	User     string `mapstructure:"user" yaml:"user"`
	Password string `mapstructure:"password" yaml:"password"`
	DBName   string `mapstructure:"dbname" yaml:"dbname"`
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) LoadConfig(configPath string) error {
	return c.loadConfigByFile(configPath)
}

func (c *Config) loadConfigByFile(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("config path is empty")
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file %q: %w", configPath, err)
	}

	if err := v.Unmarshal(c); err != nil {
		return fmt.Errorf("decode config file %q: %w", configPath, err)
	}

	return nil
}
