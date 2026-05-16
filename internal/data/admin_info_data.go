package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"

	"gorm.io/gorm"
)

var ErrAdminInfoNotFound = errors.New("admin info not found")

type AdminInfo = model.AdminInfo

type AdminInfoData struct {
	db *gorm.DB
}

func NewAdminInfoData(db *gorm.DB) *AdminInfoData {
	return &AdminInfoData{db: db}
}

// FirstActive 按 ID 升序读取第一条未删除的管理员信息。
func (a *AdminInfoData) FirstActive(ctx context.Context) (*AdminInfo, error) {
	var admin model.AdminInfo
	err := a.db.WithContext(ctx).
		Order("id ASC").
		First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAdminInfoNotFound
	}
	if err != nil {
		return nil, err
	}

	return &admin, nil
}

// Create 使用给定密码创建第一条管理员信息，密码内容由 biz 层负责提前加密。
func (a *AdminInfoData) Create(ctx context.Context, passwd string) error {
	return a.db.WithContext(ctx).Create(&model.AdminInfo{Password: passwd}).Error
}

// SetPassword 更新第一条未删除管理员的密码，密码内容由 biz 层负责提前加密。
func (a *AdminInfoData) SetPassword(ctx context.Context, passwd string) error {
	admin, err := a.FirstActive(ctx)
	if err != nil {
		return err
	}

	return a.db.WithContext(ctx).Model(&model.AdminInfo{}).
		Where("id = ?", admin.ID).
		Update("password", passwd).Error
}
