package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

type Character = model.Character

// CharacterData 负责角色表的持久化读写，所有查询都会限定在小说 ID 下。
type CharacterData struct {
	db *gorm.DB
}

// NewCharacterData 创建角色数据访问对象。
func NewCharacterData(db *gorm.DB) *CharacterData {
	return &CharacterData{db: db}
}

// Create 新增角色资料，角色始终归属于指定小说。
func (d *CharacterData) Create(ctx context.Context, character *Character) (*Character, error) {
	if err := d.db.WithContext(ctx).Create(character).Error; err != nil {
		return nil, err
	}

	return character, nil
}

// List 分页查询小说下的角色列表，只选择列表页需要展示的字段，避免正文详情被误返回。
func (d *CharacterData) List(ctx context.Context, novelID int64, offset, limit int) ([]Character, int64, error) {
	var total int64
	err := d.db.WithContext(ctx).
		Model(&model.Character{}).
		Where("novel_id = ?", novelID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var characters []Character
	err = d.db.WithContext(ctx).
		Select("name, gender, appearance_img_url, status, characters_tags").
		Where("novel_id = ?", novelID).
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&characters).Error
	if err != nil {
		return nil, 0, err
	}

	return characters, total, nil
}

// FindByID 查询指定小说下的角色详情，避免通过角色 ID 跨小说读取数据。
func (d *CharacterData) FindByID(ctx context.Context, novelID int64, characterID uint) (*Character, error) {
	var character Character
	err := d.db.WithContext(ctx).
		Where("id = ? AND novel_id = ?", characterID, novelID).
		First(&character).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.CharacterNotFound
		}
		return nil, err
	}

	return &character, nil
}

// Update 按传入字段局部更新角色资料，空更新时只校验角色存在并返回当前详情。
func (d *CharacterData) Update(ctx context.Context, novelID int64, characterID uint, values map[string]any) (*Character, error) {
	if len(values) == 0 {
		return d.FindByID(ctx, novelID, characterID)
	}

	result := d.db.WithContext(ctx).
		Model(&model.Character{}).
		Where("id = ? AND novel_id = ?", characterID, novelID).
		Updates(values)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, common.CharacterNotFound
	}

	return d.FindByID(ctx, novelID, characterID)
}

// Delete 软删除指定小说下的角色，找不到角色时返回统一业务错误。
func (d *CharacterData) Delete(ctx context.Context, novelID int64, characterID uint) error {
	result := d.db.WithContext(ctx).
		Where("id = ? AND novel_id = ?", characterID, novelID).
		Delete(&model.Character{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.CharacterNotFound
	}

	return nil
}
