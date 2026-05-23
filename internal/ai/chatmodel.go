package ai

import (
	"Novels_AI/backend/internal/data"
	"context"
	"errors"
	"log/slog"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"gorm.io/gorm"
)

func NewChatModel(ctx context.Context, db *data.AIProviderData) error {
	//现在查询启用的模型
	aiProvider, err := db.FindEnabled(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.WarnContext(ctx, "没有启用的 ai 提供商", "err", err)
			return nil
		}
		slog.ErrorContext(ctx, "查询启用中的 ai 提供商失败", "err", err)
		return err
	}
	_, err = newModle(ctx, aiProvider)
	return nil

}

func newModle(ctx context.Context, provider *data.AIProvider) (*agenticark.Model, error) {
	return agenticark.New(ctx, &agenticark.Config{})
}
