package novel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var defaultMindMapData = datatypes.JSON([]byte(`{"data":{"text":"根节点","uid":"root","expand":true},"children":[]}`))

type NovelMindMap = data.NovelMindMap

type NovelMindMapRepo interface {
	FindByNovelID(ctx context.Context, novelID int64) (*data.NovelMindMap, error)
	Save(ctx context.Context, mindMap *data.NovelMindMap) (*data.NovelMindMap, error)
}

// MindMapUseCase 负责保存 SimpleMindMap 的完整数据，并提供少量节点级操作。
//
// 设计取舍：
//  1. SimpleMindMap 的节点树、备注、概要、关联线、外框、主题、布局、视图和插件扩展字段都能落在同一份 JSON 里，
//     因此数据库只保存整图 JSONB，避免后端拆字段后跟不上前端库的数据结构变化。
//  2. 前端最简单的调用方式是“进入页面 GET 整图 -> setData -> 用户编辑 -> 点击保存 PUT 整图”。
//  3. 节点 CRUD 是可选能力，适合前端想在新增、修改、删除单个节点时立即落库；如果前端已经维护完整状态，
//     可以只使用整图保存接口，不必每次节点变动都调后端。
type MindMapUseCase struct {
	novelData   NovelRepo
	mindMapData NovelMindMapRepo
}

func NewMindMapUseCase(novelData NovelRepo, mindMapData NovelMindMapRepo) *MindMapUseCase {
	return &MindMapUseCase{novelData: novelData, mindMapData: mindMapData}
}

// GetMindMap 返回小说的思维导图；首次访问时创建一个默认根节点，方便前端直接 setData。
//
// 前端通常在打开“思维导图编辑页”时调用它，然后执行 mindMap.setData(response.mindMapData)。
func (uc *MindMapUseCase) GetMindMap(ctx context.Context, novelID int64) (*data.NovelMindMap, error) {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return nil, err
	}

	return uc.getOrCreateMindMap(ctx, novelID)
}

// SaveMindMap 保存前端通过 SimpleMindMap getData 或 getData(true) 得到的完整 JSON。
//
// 建议前端在这些时机调用：
// 1. 用户点击“保存”按钮。
// 2. 用户离开思维导图页面前，如果检测到本地有未保存变更。
// 3. 监听 SimpleMindMap 的变更事件后做防抖自动保存，例如停止编辑几秒后保存一次。
// 4. 用户批量调整布局、主题、外框、关联线或多个节点后，一次性提交整图快照。
func (uc *MindMapUseCase) SaveMindMap(ctx context.Context, params dto.SaveMindMapRequest) (*data.NovelMindMap, error) {
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}

	mindMapData, err := normalizeMindMapData(params.MindMapData)
	if err != nil {
		return nil, err
	}

	mindMap, err := uc.getOrCreateMindMap(ctx, params.NovelID)
	if err != nil {
		return nil, err
	}

	mindMap.MindMapData = mindMapData
	savedMindMap, err := uc.mindMapData.Save(ctx, mindMap)
	if err != nil {
		slog.ErrorContext(ctx, "保存小说思维导图失败", "novelId", params.NovelID, "err", err)
		return nil, err
	}

	return savedMindMap, nil
}

// GetNode 按 SimpleMindMap 节点 uid 读取单个节点 JSON。
//
// 前端一般不需要频繁调用它；如果右侧属性面板需要重新拉取某个节点的后端最新状态，可以用这个接口。
func (uc *MindMapUseCase) GetNode(ctx context.Context, novelID int64, nodeUID string) (datatypes.JSON, error) {
	mindMap, err := uc.GetMindMap(ctx, novelID)
	if err != nil {
		return nil, err
	}

	_, root, err := parseMindMapDocument(mindMap.MindMapData)
	if err != nil {
		return nil, err
	}

	node, ok := findMindMapNode(root, nodeUID)
	if !ok {
		return nil, common.MindMapNodeNotFound
	}

	return marshalMindMapJSON(node)
}

// CreateNode 在指定父节点下新增子节点，节点 uid 不传时由后端生成。
//
// 调用时机：用户点击“新增子节点/同级节点”并确认后，如果希望新增动作马上保存到数据库，就调这个接口。
// 如果前端只是先在 SimpleMindMap 本地执行新增，后续点击“保存”时提交整图，也可以不调这个接口。
func (uc *MindMapUseCase) CreateNode(ctx context.Context, params dto.CreateMindMapNodeRequest) (datatypes.JSON, error) {
	mindMap, document, root, err := uc.mutableMindMap(ctx, params.NovelID)
	if err != nil {
		return nil, err
	}

	node, err := parseMindMapNode(params.Node)
	if err != nil {
		return nil, err
	}

	uid := mindMapNodeUID(node)
	if _, ok := findMindMapNode(root, uid); ok {
		return nil, common.MindMapNodeExists
	}

	parent, ok := findMindMapNode(root, params.ParentUID)
	if !ok {
		return nil, common.MindMapNodeNotFound
	}

	insertMindMapChild(parent, node, params.Index)
	if err := uc.saveMindMapDocument(ctx, mindMap, document); err != nil {
		return nil, err
	}

	return marshalMindMapJSON(node)
}

// UpdateNode 只更新节点 data 字段，保留 children 子树，避免误删后代节点。
//
// 调用时机：用户在节点属性面板里只改了单个节点的属性并点击保存，例如文本、富文本、备注、概要、
// 标签、图片、超链接、关联线字段或自定义位置。结构性变更很多时，整图保存更简单。
func (uc *MindMapUseCase) UpdateNode(ctx context.Context, params dto.UpdateMindMapNodeRequest) (datatypes.JSON, error) {
	mindMap, document, root, err := uc.mutableMindMap(ctx, params.NovelID)
	if err != nil {
		return nil, err
	}

	node, ok := findMindMapNode(root, params.NodeUID)
	if !ok {
		return nil, common.MindMapNodeNotFound
	}

	dataMap, err := parseMindMapDataMap(params.Data)
	if err != nil {
		return nil, err
	}
	dataMap["uid"] = params.NodeUID
	node["data"] = dataMap

	if err := uc.saveMindMapDocument(ctx, mindMap, document); err != nil {
		return nil, err
	}

	return marshalMindMapJSON(node)
}

// DeleteNode 删除指定节点及其所有后代节点，符合 SimpleMindMap 的树形 children 数据结构。
//
// 调用时机：用户确认删除某个节点并希望立即落库时调用。因为 SimpleMindMap 是 children 树结构，
// 删除父节点会自然删除它下面的整棵子树，也就是用户提到的“删除节点2，节点2后面的节点一起删除”。
// 如果前端已经在画布里完成删除并稍后整图保存，也可以只调用 SaveMindMap。
func (uc *MindMapUseCase) DeleteNode(ctx context.Context, novelID int64, nodeUID string) error {
	mindMap, document, root, err := uc.mutableMindMap(ctx, novelID)
	if err != nil {
		return err
	}

	if mindMapNodeUID(root) == nodeUID {
		return common.MindMapRootDeleteNotAllowed
	}
	if !deleteMindMapNode(root, nodeUID) {
		return common.MindMapNodeNotFound
	}

	return uc.saveMindMapDocument(ctx, mindMap, document)
}

func (uc *MindMapUseCase) ensureNovelExists(ctx context.Context, novelID int64) error {
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

func (uc *MindMapUseCase) getOrCreateMindMap(ctx context.Context, novelID int64) (*data.NovelMindMap, error) {
	mindMap, err := uc.mindMapData.FindByNovelID(ctx, novelID)
	if err == nil {
		return mindMap, nil
	}
	if !errors.Is(err, common.MindMapNotFound) {
		return nil, err
	}

	return uc.mindMapData.Save(ctx, &data.NovelMindMap{
		NovelID:     novelID,
		MindMapData: append(datatypes.JSON(nil), defaultMindMapData...),
	})
}

func (uc *MindMapUseCase) mutableMindMap(ctx context.Context, novelID int64) (*data.NovelMindMap, map[string]any, map[string]any, error) {
	mindMap, err := uc.GetMindMap(ctx, novelID)
	if err != nil {
		return nil, nil, nil, err
	}

	document, root, err := parseMindMapDocument(mindMap.MindMapData)
	if err != nil {
		return nil, nil, nil, err
	}

	return mindMap, document, root, nil
}

func (uc *MindMapUseCase) saveMindMapDocument(ctx context.Context, mindMap *data.NovelMindMap, document map[string]any) error {
	raw, err := marshalMindMapJSON(document)
	if err != nil {
		return err
	}

	mindMap.MindMapData = raw
	if _, err := uc.mindMapData.Save(ctx, mindMap); err != nil {
		slog.ErrorContext(ctx, "保存小说思维导图失败", "novelId", mindMap.NovelID, "err", err)
		return err
	}

	return nil
}

func normalizeMindMapData(raw json.RawMessage) (datatypes.JSON, error) {
	document, _, err := parseMindMapDocument(raw)
	if err != nil {
		return nil, err
	}

	return marshalMindMapJSON(document)
}

func parseMindMapDocument(raw []byte) (map[string]any, map[string]any, error) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, nil, common.InvalidRequest
	}

	// getData(true) 通常会返回包含 root/theme/layout/view 的对象，root 才是真正的节点树。
	if rootValue, ok := document["root"]; ok {
		root, ok := rootValue.(map[string]any)
		if !ok {
			return nil, nil, common.InvalidRequest
		}
		normalizeMindMapNode(root)
		return document, root, nil
	}

	// getData(false) 或前端只传 setData 所需节点树时，顶层对象自身就是根节点。
	if _, ok := document["data"]; !ok {
		return nil, nil, common.InvalidRequest
	}

	normalizeMindMapNode(document)
	return document, document, nil
}

func parseMindMapNode(raw json.RawMessage) (map[string]any, error) {
	var node map[string]any
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, common.InvalidRequest
	}

	normalizeMindMapNode(node)
	if mindMapNodeUID(node) == "" {
		dataMap, _ := node["data"].(map[string]any)
		dataMap["uid"] = fmt.Sprintf("node_%d", time.Now().UnixNano())
	}

	return node, nil
}

func parseMindMapDataMap(raw json.RawMessage) (map[string]any, error) {
	var dataMap map[string]any
	if err := json.Unmarshal(raw, &dataMap); err != nil {
		return nil, common.InvalidRequest
	}

	return dataMap, nil
}

func normalizeMindMapNode(node map[string]any) {
	dataMap, ok := node["data"].(map[string]any)
	if !ok {
		dataMap = map[string]any{}
		node["data"] = dataMap
	}

	if _, ok := node["children"].([]any); !ok {
		node["children"] = []any{}
	}
}

func mindMapNodeUID(node map[string]any) string {
	dataMap, ok := node["data"].(map[string]any)
	if !ok {
		return ""
	}

	uid, _ := dataMap["uid"].(string)
	return strings.TrimSpace(uid)
}

func findMindMapNode(root map[string]any, uid string) (map[string]any, bool) {
	if mindMapNodeUID(root) == uid {
		return root, true
	}

	children, _ := root["children"].([]any)
	for _, child := range children {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}

		if found, ok := findMindMapNode(childMap, uid); ok {
			return found, true
		}
	}

	return nil, false
}

func insertMindMapChild(parent map[string]any, child map[string]any, index *int) {
	children, _ := parent["children"].([]any)
	if index == nil || *index < 0 || *index >= len(children) {
		parent["children"] = append(children, child)
		return
	}

	insertAt := *index
	children = append(children, nil)
	copy(children[insertAt+1:], children[insertAt:])
	children[insertAt] = child
	parent["children"] = children
}

func deleteMindMapNode(root map[string]any, uid string) bool {
	children, _ := root["children"].([]any)
	nextChildren := make([]any, 0, len(children))
	deleted := false
	for _, child := range children {
		childMap, ok := child.(map[string]any)
		if !ok {
			nextChildren = append(nextChildren, child)
			continue
		}

		if mindMapNodeUID(childMap) == uid {
			deleted = true
			continue
		}

		if deleteMindMapNode(childMap, uid) {
			deleted = true
		}
		nextChildren = append(nextChildren, childMap)
	}

	root["children"] = nextChildren
	return deleted
}

func marshalMindMapJSON(value any) (datatypes.JSON, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return datatypes.JSON(raw), nil
}
