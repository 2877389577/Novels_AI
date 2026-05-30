package bootstrap

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(config PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		config.Host,
		config.User,
		config.Password,
		config.DBName,
		config.Port,
	)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
		// pgx 默认会启用隐式 prepared statement 缓存，在 PgBouncer 或部分代理/连接复用场景下，
		// 可能因为服务端已存在同名 statement 而返回 SQLSTATE 42P05，这里改用简单协议避免该类冲突。
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect postgres database: %w", err)
	}
	slog.Info("connect postgres database success")

	if err := AutoMigrateDB(db); err != nil {
		return nil, err
	}

	return db, nil
}
