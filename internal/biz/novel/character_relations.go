package novel

import (
	"context"
	"errors"
	"log/slog"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CharacterRelationNode = data.CharacterRelationNode
type CharacterRelation = data.CharacterRelation

// CharacterRelationUseCase 承载角色关系图的业务规则，包括小说归属和角色归属校验。
type CharacterRelationUseCase struct {
	novelData     NovelRepo
	characterData CharacterRepo
	relationData  CharacterRelationRepo
}

// CharacterRelationRepo 定义角色关系图业务层需要的数据访问能力。
type CharacterRelationRepo interface {
	ListCharacters(ctx context.Context, novelID int64) ([]data.Character, error)
	ListNodes(ctx context.Context, novelID int64) ([]data.CharacterRelationNode, error)
	SaveNodeLayouts(ctx context.Context, novelID int64, nodes []data.CharacterRelationNode) ([]data.CharacterRelationNode, error)
	ListEnabledRelations(ctx context.Context, novelID int64) ([]data.CharacterRelation, error)
	ListRelations(ctx context.Context, novelID int64) ([]data.CharacterRelation, int64, error)
	FindRelationByID(ctx context.Context, novelID int64, relationID int64) (*data.CharacterRelation, error)
	RelationExists(ctx context.Context, novelID, sourceCharacterID, targetCharacterID int64, relationType string, excludeID int64) (bool, error)
	CreateRelation(ctx context.Context, relation *data.CharacterRelation) (*data.CharacterRelation, error)
	UpdateRelation(ctx context.Context, relation *data.CharacterRelation) (*data.CharacterRelation, error)
	DeleteRelation(ctx context.Context, novelID int64, relationID int64) error
}

// CharacterRelationGraphResult 包含前端绘制 Vue Flow 关系图需要的角色、布局和关系边原始数据。
type CharacterRelationGraphResult struct {
	Characters []data.Character
	Nodes      []data.CharacterRelationNode
	Relations  []data.CharacterRelation
}

// ListCharacterRelationResult 包含角色关系管理列表和总数。
type ListCharacterRelationResult struct {
	Items []data.CharacterRelation
	Total int64
}

// NewCharacterRelationUseCase 创建角色关系图业务用例。
func NewCharacterRelationUseCase(novelData NovelRepo, characterData CharacterRepo, relationData CharacterRelationRepo) *CharacterRelationUseCase {
	return &CharacterRelationUseCase{
		novelData:     novelData,
		characterData: characterData,
		relationData:  relationData,
	}
}

// GetCharacterRelationGraph 查询指定小说下的角色关系图数据。
func (uc *CharacterRelationUseCase) GetCharacterRelationGraph(ctx context.Context, novelID int64) (*CharacterRelationGraphResult, error) {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return nil, err
	}

	characters, err := uc.relationData.ListCharacters(ctx, novelID)
	if err != nil {
		return nil, err
	}

	nodes, err := uc.relationData.ListNodes(ctx, novelID)
	if err != nil {
		return nil, err
	}

	relations, err := uc.relationData.ListEnabledRelations(ctx, novelID)
	if err != nil {
		return nil, err
	}

	return &CharacterRelationGraphResult{
		Characters: characters,
		Nodes:      nodes,
		Relations:  relations,
	}, nil
}

// SaveCharacterRelationNodeLayouts 批量保存角色关系图节点布局。
func (uc *CharacterRelationUseCase) SaveCharacterRelationNodeLayouts(ctx context.Context, params dto.SaveCharacterRelationNodeLayoutsRequest) ([]data.CharacterRelationNode, error) {
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}

	nodes := make([]data.CharacterRelationNode, 0, len(params.Nodes))
	for _, item := range params.Nodes {
		if err := uc.ensureCharacterBelongsToNovel(ctx, params.NovelID, item.CharacterID); err != nil {
			return nil, err
		}

		nodeType := item.NodeType
		if nodeType == "" {
			nodeType = "character"
		}

		status := int16(1)
		if item.Status != nil {
			status = *item.Status
		}

		nodes = append(nodes, data.CharacterRelationNode{
			NovelID:     params.NovelID,
			CharacterID: item.CharacterID,
			NodeType:    nodeType,
			PositionX:   item.PositionX,
			PositionY:   item.PositionY,
			Width:       item.Width,
			Height:      item.Height,
			Hidden:      item.Hidden,
			Locked:      item.Locked,
			Style:       normalizeJSONField(item.Style),
			ExtraData:   normalizeJSONField(item.ExtraData),
			Status:      status,
		})
	}

	return uc.relationData.SaveNodeLayouts(ctx, params.NovelID, nodes)
}

// ListCharacterRelations 查询指定小说下的所有未删除角色关系。
func (uc *CharacterRelationUseCase) ListCharacterRelations(ctx context.Context, novelID int64) (*ListCharacterRelationResult, error) {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return nil, err
	}

	items, total, err := uc.relationData.ListRelations(ctx, novelID)
	if err != nil {
		return nil, err
	}

	return &ListCharacterRelationResult{
		Items: items,
		Total: total,
	}, nil
}

// CreateCharacterRelation 新增角色关系，并确保关系两端都属于当前小说。
func (uc *CharacterRelationUseCase) CreateCharacterRelation(ctx context.Context, params dto.CreateCharacterRelationRequest) (*data.CharacterRelation, error) {
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}
	if err := uc.ensureRelationCharacters(ctx, params.NovelID, params.SourceCharacterID, params.TargetCharacterID); err != nil {
		return nil, err
	}

	exists, err := uc.relationData.RelationExists(ctx, params.NovelID, params.SourceCharacterID, params.TargetCharacterID, params.RelationType, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, common.CharacterRelationExists
	}

	direction := int16(2)
	if params.Direction != nil {
		direction = *params.Direction
	}

	edgeType := params.EdgeType
	if edgeType == "" {
		edgeType = "default"
	}

	status := int16(1)
	if params.Status != nil {
		status = *params.Status
	}

	return uc.relationData.CreateRelation(ctx, &data.CharacterRelation{
		NovelID:           params.NovelID,
		SourceCharacterID: params.SourceCharacterID,
		TargetCharacterID: params.TargetCharacterID,
		RelationType:      params.RelationType,
		RelationLabel:     params.RelationLabel,
		Description:       params.Description,
		Direction:         direction,
		EdgeType:          edgeType,
		Animated:          params.Animated,
		SourceHandle:      params.SourceHandle,
		TargetHandle:      params.TargetHandle,
		SortOrder:         params.SortOrder,
		Style:             normalizeJSONField(params.Style),
		ExtraData:         normalizeJSONField(params.ExtraData),
		Status:            status,
	})
}

// GetCharacterRelation 查询指定小说下的角色关系详情。
func (uc *CharacterRelationUseCase) GetCharacterRelation(ctx context.Context, novelID int64, relationID int64) (*data.CharacterRelation, error) {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return nil, err
	}

	return uc.relationData.FindRelationByID(ctx, novelID, relationID)
}

// UpdateCharacterRelation 全量更新角色关系。
func (uc *CharacterRelationUseCase) UpdateCharacterRelation(ctx context.Context, params dto.UpdateCharacterRelationRequest) (*data.CharacterRelation, error) {
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}
	if _, err := uc.relationData.FindRelationByID(ctx, params.NovelID, params.RelationID); err != nil {
		return nil, err
	}
	if err := uc.ensureRelationCharacters(ctx, params.NovelID, params.SourceCharacterID, params.TargetCharacterID); err != nil {
		return nil, err
	}

	exists, err := uc.relationData.RelationExists(ctx, params.NovelID, params.SourceCharacterID, params.TargetCharacterID, params.RelationType, params.RelationID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, common.CharacterRelationExists
	}

	edgeType := params.EdgeType
	if edgeType == "" {
		edgeType = "default"
	}

	return uc.relationData.UpdateRelation(ctx, &data.CharacterRelation{
		ID:                params.RelationID,
		NovelID:           params.NovelID,
		SourceCharacterID: params.SourceCharacterID,
		TargetCharacterID: params.TargetCharacterID,
		RelationType:      params.RelationType,
		RelationLabel:     params.RelationLabel,
		Description:       params.Description,
		Direction:         params.Direction,
		EdgeType:          edgeType,
		Animated:          params.Animated,
		SourceHandle:      params.SourceHandle,
		TargetHandle:      params.TargetHandle,
		SortOrder:         params.SortOrder,
		Style:             normalizeJSONField(params.Style),
		ExtraData:         normalizeJSONField(params.ExtraData),
		Status:            params.Status,
	})
}

// DeleteCharacterRelation 物理删除指定小说下的角色关系。
func (uc *CharacterRelationUseCase) DeleteCharacterRelation(ctx context.Context, novelID int64, relationID int64) error {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return err
	}

	return uc.relationData.DeleteRelation(ctx, novelID, relationID)
}

// ensureCharacterBelongsToNovel 确认角色属于当前小说，避免跨小说保存布局或关系。
func (uc *CharacterRelationUseCase) ensureCharacterBelongsToNovel(ctx context.Context, novelID int64, characterID int64) error {
	_, err := uc.characterData.FindByID(ctx, novelID, uint(characterID))
	return err
}

// ensureRelationCharacters 确认关系两端合法且都属于当前小说。
func (uc *CharacterRelationUseCase) ensureRelationCharacters(ctx context.Context, novelID, sourceCharacterID, targetCharacterID int64) error {
	if sourceCharacterID == targetCharacterID {
		return common.CharacterRelationSelfNotAllowed
	}
	if err := uc.ensureCharacterBelongsToNovel(ctx, novelID, sourceCharacterID); err != nil {
		return err
	}

	return uc.ensureCharacterBelongsToNovel(ctx, novelID, targetCharacterID)
}

// ensureNovelExists 确认角色关系图归属的小说存在，避免给不存在的小说维护关系数据。
func (uc *CharacterRelationUseCase) ensureNovelExists(ctx context.Context, novelID int64) error {
	_, err := uc.novelData.FindByID(ctx, uint(novelID))
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNovelNotFound) {
		return ErrNovelNotFound
	}

	slog.ErrorContext(ctx, "查询小说失败", "err", err)
	return err
}

// normalizeJSONField 将未传、空值和 null 统一整理为 jsonb 默认对象。
func normalizeJSONField(field dto.JSONField) datatypes.JSON {
	if !field.Set || len(field.Value) == 0 || string(field.Value) == "null" {
		return datatypes.JSON([]byte("{}"))
	}

	return field.Value
}
