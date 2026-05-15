package main

import (
	"Novels_AI/backend/internal/bootstrap"
	"flag"
	"log"
	"log/slog"
	"os"
)

func main() {
	// 读取命令行传入的配置文件路径；如果没有传入，则使用项目默认配置文件。
	configPath := flag.String("config", "config/config.yaml", "config file path")
	flag.Parse()

	// 加载配置
	config := bootstrap.NewConfig()
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	// 在其他基础设施初始化前先初始化日志服务。
	// 这样后续数据库、HTTP 服务等模块启动失败时，都可以通过 slog 输出结构化日志。
	logger, err := bootstrap.NewLogger(config.Log)
	if err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	// 将当前 logger 设置为 slog 的全局默认 logger。
	// 后续代码可以直接使用 slog.Info/slog.Error 等包级函数，不需要层层传递 logger。
	slog.SetDefault(logger)

	// 初始化数据库
	if _, err = bootstrap.NewDB(config.System.Postgres); err != nil {
		slog.Error("init database failed", "error", err)
		os.Exit(1)
	}
}
