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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect postgres database: %w", err)
	}
	slog.Info("connect postgres database success")

	if err := AutoMigrateDB(db); err != nil {
		return nil, err
	}

	return db, nil
}
