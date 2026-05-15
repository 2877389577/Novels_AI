package bootstrap

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server" yaml:"server"`
	System SystemConfig `mapstructure:"system" yaml:"system"`
	Log    LogConfig    `mapstructure:"log" yaml:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port" yaml:"port"`
	Host string `mapstructure:"host" yaml:"host"`
}

type LogConfig struct {
	Output string        `mapstructure:"output" yaml:"output"`
	Level  string        `mapstructure:"level" yaml:"level"`
	File   LogFileConfig `mapstructure:"file" yaml:"file"`
}

type LogFileConfig struct {
	Path       string `mapstructure:"path" yaml:"path"`
	MaxSizeMB  int64  `mapstructure:"max_size_mb" yaml:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups" yaml:"max_backups"`
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
