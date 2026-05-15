package main

import (
	"Novels_AI/backend/internal/bootstrap"
	"flag"
	"log"
	"log/slog"
	"os"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "config file path")
	flag.Parse()

	// 加载配置
	config := bootstrap.NewConfig()
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	logger, err := bootstrap.NewLogger(config.Log)
	if err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	slog.SetDefault(logger)

	// 初始化数据库
	if _, err = bootstrap.NewDB(config.System.Postgres); err != nil {
		slog.Error("init database failed", "error", err)
		os.Exit(1)
	}
}
