package upload

import (
	"context"

	"Novels_AI/backend/internal/data"
)

type UploadObject = data.UploadObject
type UploadedObject = data.UploadedObject

type UploadUseCase struct {
	uploadData UploadRepo
}

type UploadRepo interface {
	Upload(ctx context.Context, object data.UploadObject) (*data.UploadedObject, error)
}

func NewUploadUseCase(uploadData UploadRepo) *UploadUseCase {
	return &UploadUseCase{uploadData: uploadData}
}

// Upload 负责上传业务编排，实际存储细节由 data 层实现。
func (uc *UploadUseCase) Upload(ctx context.Context, object UploadObject) (*UploadedObject, error) {
	return uc.uploadData.Upload(ctx, object)
}
