package ai

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
)

func NewChatModel(ctx context.Context, baseURL, modelName, apiKey string) (*agenticark.Model, error) {
	//现在查询启用的模型

	return agenticark.New(ctx, &agenticark.Config{
		Model:   modelName,
		BaseURL: baseURL,
		APIKey:  apiKey,
	})
}
