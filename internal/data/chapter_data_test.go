package data

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestChapterFindByIDQueryUsesChapterIDAndNovelID(t *testing.T) {
	db := newDryRunPostgresDB(t)
	repo := NewChapterData(db)

	tx := repo.chapterFindByIDQuery(context.Background(), 4, 3).First(&Chapter{})

	// 章节查询必须同时限定章节 ID 和小说 ID，否则修改章节后发布的 AI 总结事件会拿到其他小说的章节。
	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "id =") || !strings.Contains(sql, "novel_id =") {
		t.Fatalf("expected query to include id and novel_id conditions, got SQL: %s", sql)
	}

	if len(tx.Statement.Vars) < 2 {
		t.Fatalf("expected query vars to include chapter ID and novel ID, got %#v", tx.Statement.Vars)
	}
	if tx.Statement.Vars[0] != uint(3) || tx.Statement.Vars[1] != int64(4) {
		t.Fatalf("unexpected query vars: %#v", tx.Statement.Vars)
	}
}

func TestChapterFindByIDQueryAllowsOnlyChapterIDWhenNovelIDEmpty(t *testing.T) {
	db := newDryRunPostgresDB(t)
	repo := NewChapterData(db)

	tx := repo.chapterFindByIDQuery(context.Background(), 0, 3).First(&Chapter{})

	// 角色模块仍会按章节 ID 单独查询章节，这里保留原有兼容行为。
	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "id =") {
		t.Fatalf("expected query to include id condition, got SQL: %s", sql)
	}
	if strings.Contains(sql, "novel_id =") {
		t.Fatalf("expected query to skip novel_id condition, got SQL: %s", sql)
	}

	if len(tx.Statement.Vars) < 1 || tx.Statement.Vars[0] != uint(3) {
		t.Fatalf("unexpected query vars: %#v", tx.Statement.Vars)
	}
}

func newDryRunPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=test dbname=test sslmode=disable",
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("open dry run db: %v", err)
	}

	return db
}
