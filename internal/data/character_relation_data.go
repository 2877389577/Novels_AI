package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

type CharacterRelationNode = model.CharacterRelationNode
type CharacterRelation = model.CharacterRelation

// CharacterRelationData 负责角色关系图节点布局和角色关系边的持久化读写。
type CharacterRelationData struct {
	db *gorm.DB
}

// NewCharacterRelationData 创建角色关系图数据访问对象。
func NewCharacterRelationData(db *gorm.DB) *CharacterRelationData {
	return &CharacterRelationData{db: db}
}

// ListCharacters 查询小说下所有未删除角色，用于组装关系图节点。
func (d *CharacterRelationData) ListCharacters(ctx context.Context, novelID int64) ([]Character, error) {
	var characters []Character
	err := d.db.WithContext(ctx).
		Select("id, novel_id, name, gender, appearance_img_url, status, characters_tags").
		Where("novel_id = ?", novelID).
		Order("id ASC").
		Find(&characters).Error
	if err != nil {
		return nil, err
	}

	return characters, nil
}

// ListNodes 查询小说下已保存的节点布局，未保存布局的角色由业务层或 service 层使用默认坐标。
func (d *CharacterRelationData) ListNodes(ctx context.Context, novelID int64) ([]CharacterRelationNode, error) {
	var nodes []CharacterRelationNode
	err := d.db.WithContext(ctx).
		Where("novel_id = ?", novelID).
		Order("id ASC").
		Find(&nodes).Error
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// SaveNodeLayouts 按 novel_id + character_id 保存节点布局，已有布局全量覆盖，不存在则新增。
func (d *CharacterRelationData) SaveNodeLayouts(ctx context.Context, novelID int64, nodes []CharacterRelationNode) ([]CharacterRelationNode, error) {
	savedNodes := make([]CharacterRelationNode, 0, len(nodes))
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, node := range nodes {
			node.NovelID = novelID

			var current CharacterRelationNode
			err := tx.Where("novel_id = ? AND character_id = ?", novelID, node.CharacterID).
				First(&current).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&node).Error; err != nil {
					return err
				}
				savedNodes = append(savedNodes, node)
				continue
			}
			if err != nil {
				return err
			}

			// 只覆盖布局和展示字段，保留当前记录的主键和创建时间。
			current.NodeType = node.NodeType
			current.PositionX = node.PositionX
			current.PositionY = node.PositionY
			current.Width = node.Width
			current.Height = node.Height
			current.Hidden = node.Hidden
			current.Locked = node.Locked
			current.Style = node.Style
			current.ExtraData = node.ExtraData
			current.Status = node.Status
			if err := tx.Save(&current).Error; err != nil {
				return err
			}
			savedNodes = append(savedNodes, current)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return savedNodes, nil
}

// ListEnabledRelations 查询小说下启用的角色关系边，用于关系图展示。
func (d *CharacterRelationData) ListEnabledRelations(ctx context.Context, novelID int64) ([]CharacterRelation, error) {
	var relations []CharacterRelation
	err := d.db.WithContext(ctx).
		Where("novel_id = ? AND status = ?", novelID, int16(1)).
		Order("sort_order ASC, id ASC").
		Find(&relations).Error
	if err != nil {
		return nil, err
	}

	return relations, nil
}

// ListRelations 查询小说下所有关系，包含启用和停用状态，供管理页使用。
func (d *CharacterRelationData) ListRelations(ctx context.Context, novelID int64) ([]CharacterRelation, int64, error) {
	var total int64
	err := d.db.WithContext(ctx).
		Model(&model.CharacterRelation{}).
		Where("novel_id = ?", novelID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var relations []CharacterRelation
	err = d.db.WithContext(ctx).
		Where("novel_id = ?", novelID).
		Order("sort_order ASC, id ASC").
		Find(&relations).Error
	if err != nil {
		return nil, 0, err
	}

	return relations, total, nil
}

// FindRelationByID 查询指定小说下的单条角色关系，避免跨小说读取关系数据。
func (d *CharacterRelationData) FindRelationByID(ctx context.Context, novelID int64, relationID int64) (*CharacterRelation, error) {
	var relation CharacterRelation
	err := d.db.WithContext(ctx).
		Where("id = ? AND novel_id = ?", relationID, novelID).
		First(&relation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.CharacterRelationNotFound
		}
		return nil, err
	}

	return &relation, nil
}

// RelationExists 检查同一本小说下是否已经存在相同起点、终点和类型的有效关系。
func (d *CharacterRelationData) RelationExists(ctx context.Context, novelID, sourceCharacterID, targetCharacterID int64, relationType string, excludeID int64) (bool, error) {
	query := d.db.WithContext(ctx).
		Model(&model.CharacterRelation{}).
		Where("novel_id = ? AND source_character_id = ? AND target_character_id = ? AND relation_type = ?",
			novelID, sourceCharacterID, targetCharacterID, relationType)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return false, err
	}

	return total > 0, nil
}

// CreateRelation 新增角色关系，数据库唯一索引会兜底保护重复有效关系。
func (d *CharacterRelationData) CreateRelation(ctx context.Context, relation *CharacterRelation) (*CharacterRelation, error) {
	if err := d.db.WithContext(ctx).Create(relation).Error; err != nil {
		return nil, err
	}

	return relation, nil
}

// UpdateRelation 全量保存角色关系，并保留当前记录的主键和创建时间。
func (d *CharacterRelationData) UpdateRelation(ctx context.Context, relation *CharacterRelation) (*CharacterRelation, error) {
	current, err := d.FindRelationByID(ctx, relation.NovelID, relation.ID)
	if err != nil {
		return nil, err
	}

	current.SourceCharacterID = relation.SourceCharacterID
	current.TargetCharacterID = relation.TargetCharacterID
	current.RelationType = relation.RelationType
	current.RelationLabel = relation.RelationLabel
	current.Description = relation.Description
	current.Direction = relation.Direction
	current.EdgeType = relation.EdgeType
	current.Animated = relation.Animated
	current.SourceHandle = relation.SourceHandle
	current.TargetHandle = relation.TargetHandle
	current.SortOrder = relation.SortOrder
	current.Style = relation.Style
	current.ExtraData = relation.ExtraData
	current.Status = relation.Status
	if err := d.db.WithContext(ctx).Save(current).Error; err != nil {
		return nil, err
	}

	return d.FindRelationByID(ctx, relation.NovelID, relation.ID)
}

// DeleteRelation 物理删除指定小说下的角色关系。
func (d *CharacterRelationData) DeleteRelation(ctx context.Context, novelID int64, relationID int64) error {
	result := d.db.WithContext(ctx).
		Where("id = ? AND novel_id = ?", relationID, novelID).
		Delete(&model.CharacterRelation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.CharacterRelationNotFound
	}

	return nil
}
