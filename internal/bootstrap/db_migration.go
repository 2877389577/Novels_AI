package bootstrap

import (
	"fmt"
	"log/slog"
	"strings"

	"Novels_AI/backend/internal/data/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type migrationModel struct {
	name  string
	value any
}

// AutoMigrateDB 在服务启动时检查并同步数据库结构。
//
// GORM AutoMigrate 是非破坏性迁移：会创建缺失表、索引和字段，但不会删除已有字段或已有数据。
func AutoMigrateDB(db *gorm.DB) error {
	models := databaseMigrationModels()
	missingTables := missingMigrationTables(db, models)
	if len(missingTables) > 0 {
		slog.Info("检测到缺失数据库表，开始自动迁移", "tables", missingTables)
	}
	if len(missingTables) == 0 {
		slog.Info("开始检查数据库结构迁移")
	}

	values := make([]any, 0, len(models))
	for _, item := range models {
		values = append(values, item.value)
	}

	if err := db.AutoMigrate(values...); err != nil {
		return fmt.Errorf("auto migrate database schema: %w", err)
	}

	if err := migrateChapterUniqueIndex(db); err != nil {
		return fmt.Errorf("migrate chapter unique index: %w", err)
	}

	if err := seedDefaultAITaskConfigs(db); err != nil {
		return fmt.Errorf("seed default ai task configs: %w", err)
	}

	slog.Info("数据库结构迁移检查完成")
	return nil
}

// databaseMigrationModels 集中维护启动迁移需要覆盖的业务表模型。
func databaseMigrationModels() []migrationModel {
	return []migrationModel{
		{name: "admin_info", value: &model.AdminInfo{}},
		{name: "ai_providers", value: &model.AIProvider{}},
		{name: "ai_task_configs", value: &model.AITaskConfig{}},
		{name: "image_ai_providers", value: &model.ImageAIProvider{}},
		{name: "novels", value: &model.Novel{}},
		{name: "chapters", value: &model.Chapter{}},
		{name: "chapter_plot_analyses", value: &model.ChapterPlotAnalysis{}},
		{name: "novel_mind_maps", value: &model.NovelMindMap{}},
		{name: "characters", value: &model.Character{}},
		{name: "character_relation_nodes", value: &model.CharacterRelationNode{}},
		{name: "character_relations", value: &model.CharacterRelation{}},
	}
}

// missingMigrationTables 返回当前数据库中尚未创建的表名，用于启动日志定位迁移原因。
func missingMigrationTables(db *gorm.DB, models []migrationModel) []string {
	missingTables := make([]string, 0)
	for _, item := range models {
		if db.Migrator().HasTable(item.value) {
			continue
		}

		missingTables = append(missingTables, item.name)
	}

	return missingTables
}

// migrateChapterUniqueIndex 把旧的全量唯一约束迁移为部分唯一索引，避免软删除章节继续占用原章节编号。
func migrateChapterUniqueIndex(db *gorm.DB) error {
	ready, err := activeChapterUniqueIndexReady(db)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}

	if err := db.Exec(`ALTER TABLE chapters DROP CONSTRAINT IF EXISTS uk_novel_chapter_no`).Error; err != nil {
		return fmt.Errorf("drop old chapter unique constraint: %w", err)
	}
	if err := db.Exec(`DROP INDEX IF EXISTS uk_novel_chapter_no`).Error; err != nil {
		return fmt.Errorf("drop old chapter unique index: %w", err)
	}
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS uk_novel_chapter_no
		ON chapters (novel_id, chapter_no)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("create active chapter unique index: %w", err)
	}

	return nil
}

// activeChapterUniqueIndexReady 判断当前索引是否已经只约束未删除章节，避免每次启动都重建索引。
func activeChapterUniqueIndexReady(db *gorm.DB) (bool, error) {
	var constraintCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE c.conname = ? AND t.relname = ? AND n.nspname = ANY (current_schemas(false))
	`, "uk_novel_chapter_no", "chapters").Scan(&constraintCount).Error; err != nil {
		return false, fmt.Errorf("check old chapter unique constraint: %w", err)
	}
	if constraintCount > 0 {
		return false, nil
	}

	var indexDef string
	if err := db.Raw(`
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = ANY (current_schemas(false)) AND tablename = ? AND indexname = ?
		LIMIT 1
	`, "chapters", "uk_novel_chapter_no").Scan(&indexDef).Error; err != nil {
		return false, fmt.Errorf("check chapter unique index: %w", err)
	}

	return isActiveChapterUniqueIndex(indexDef), nil
}

// isActiveChapterUniqueIndex 用索引定义识别目标状态：同一本小说下未删除章节的 chapter_no 唯一。
func isActiveChapterUniqueIndex(indexDef string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(indexDef), " "))

	return strings.Contains(normalized, "create unique index") &&
		strings.Contains(normalized, "(novel_id, chapter_no)") &&
		strings.Contains(normalized, "deleted_at is null")
}

// seedDefaultAITaskConfigs 补齐系统内置主动 AI 任务配置，不覆盖管理员已经修改过的开关状态。
func seedDefaultAITaskConfigs(db *gorm.DB) error {
	configs := []model.AITaskConfig{
		{
			TaskCode:    model.AITaskCodeChapterPlotAnalysis,
			TaskName:    model.AITaskNameChapterPlotAnalysis,
			Description: model.AITaskDescriptionChapterPlotAnalysis,
			IsEnabled:   false,
		},
	}

	return db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "task_code"}},
			DoNothing: true,
		}).
		// 明确选择 is_enabled，避免 GORM 因默认值标签省略 false，虽然数据库默认值同样为 false。
		Select("TaskCode", "TaskName", "Description", "IsEnabled").
		Create(&configs).Error
}
