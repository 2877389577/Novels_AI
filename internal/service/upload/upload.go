package upload

import (
	"context"
	"net/http"

	uploadbiz "Novels_AI/backend/internal/biz/upload"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

type UploadService struct {
	useCase UploadUseCase
}

type UploadUseCase interface {
	Upload(ctx context.Context, object uploadbiz.UploadObject) (*uploadbiz.UploadedObject, error)
}

type uploadResponse struct {
	// Key 文件在 S3/对象存储桶里的对象路径，相当于存储内部的文件名
	Key string `json:"key"`
	// URL 前端可以直接访问的公开地址，是用配置里的 public_base_url 加上 key 拼出来的
	URL string `json:"url"`
}

func NewUploadService(useCase UploadUseCase) *UploadService {
	return &UploadService{useCase: useCase}
}

// Upload 上传文件
// @Summary 上传文件
// @Description 使用 S3 兼容对象存储上传文件，返回公开访问地址
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文件"
// @Success 200 {object} common.Response{data=uploadResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /upload [post]
func (service *UploadService) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		_ = c.Error(err)
		return
	}
	defer file.Close()

	result, err := service.useCase.Upload(c.Request.Context(), uploadbiz.UploadObject{
		FileName:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Size:        fileHeader.Size,
		Body:        file,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: &uploadResponse{
			Key: result.Key,
			URL: result.URL,
		},
	})
}
