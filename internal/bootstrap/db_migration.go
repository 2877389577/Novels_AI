package bootstrap

import (
	"fmt"
	"log/slog"

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
