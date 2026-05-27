package novel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"Novels_AI/backend/internal/ai/ai_tools"
	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/event"
	"Novels_AI/backend/internal/pkg/common"
)

func TestEnabledAIProviderDefaultModel(t *testing.T) {
	// 自动章节剧情总结必须使用启用 Provider 的 default_model，而不是模型列表里的第一个模型。
	modelName, ok := enabledAIProviderDefaultModel(&data.AIProvider{DefaultModel: "  doubao-test  "})
	if !ok {
		t.Fatal("expected default model to be selected")
	}
	if modelName != "doubao-test" {
		t.Fatalf("expected doubao-test, got %q", modelName)
	}
}

func TestEnabledAIProviderDefaultModelReturnsFalseWhenEmpty(t *testing.T) {
	if _, ok := enabledAIProviderDefaultModel(&data.AIProvider{DefaultModel: "   "}); ok {
		t.Fatal("expected empty default model to return false")
	}
}

func TestCreateChapterPublishesChapterSavedEvent(t *testing.T) {
	bus := event.NewBus()
	var got ChapterSavedEvent
	bus.Subscribe(chapterSavedEventName, func(ctx context.Context, evt event.Event) error {
		payload, ok := evt.Payload.(ChapterSavedEvent)
		if !ok {
			t.Fatalf("unexpected event payload: %#v", evt.Payload)
		}
		got = payload
		return nil
	})

	uc := NewChapterUseCase(fakeNovelRepo{}, &fakeChapterRepo{}, bus)
	chapter, err := uc.CreateChapter(context.Background(), dto.CreateChapterRequest{
		NovelID:   100,
		ChapterNo: 1,
		Title:     "第一章",
		Content:   "雨夜相逢。",
		WordCount: 4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chapter.ID == 0 {
		t.Fatal("expected fake chapter repo to assign chapter ID")
	}
	if got.NovelID != 100 || got.ChapterID != chapter.ID || got.Title != "第一章" || got.Content != "雨夜相逢。" {
		t.Fatalf("unexpected event payload: %#v", got)
	}
}

func TestCreateChapterIgnoresChapterSavedEventError(t *testing.T) {
	bus := event.NewBus()
	bus.Subscribe(chapterSavedEventName, func(ctx context.Context, evt event.Event) error {
		return errors.New("event failed")
	})

	uc := NewChapterUseCase(fakeNovelRepo{}, &fakeChapterRepo{}, bus)
	if _, err := uc.CreateChapter(context.Background(), dto.CreateChapterRequest{
		NovelID:   100,
		ChapterNo: 1,
		Title:     "第一章",
		Content:   "雨夜相逢。",
		WordCount: 4,
	}); err != nil {
		t.Fatalf("expected chapter save to ignore event error, got %v", err)
	}
}

func TestCreateChapterDoesNotWaitForAsyncPlotAnalysis(t *testing.T) {
	bus := event.NewBus()
	started := make(chan struct{})
	release := make(chan struct{})
	saved := make(chan struct{})
	plotRepo := &fakeChapterPlotAnalysisRepo{savedCh: saved}
	plotUseCase := &ChapterPlotAnalysisUseCase{
		analysisData: plotRepo,
		analyzer: fakeChapterPlotAnalyzer{
			started: started,
			release: release,
			result: &ai_tools.ChapterPlotAnalysisTool{
				Summary: "陆沉救下宋玉。",
			},
		},
	}
	RegisterChapterPlotAnalysisEventHandlers(bus, plotUseCase)

	uc := NewChapterUseCase(fakeNovelRepo{}, &fakeChapterRepo{}, bus)
	done := make(chan error, 1)
	go func() {
		_, err := uc.CreateChapter(context.Background(), dto.CreateChapterRequest{
			NovelID:   100,
			ChapterNo: 1,
			Title:     "第一章",
			Content:   "雨夜相逢。",
			WordCount: 4,
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			close(release)
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		close(release)
		t.Fatal("expected chapter save to return without waiting for plot analysis")
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("expected async plot analysis to start")
	}

	close(release)
	select {
	case <-saved:
	case <-time.After(time.Second):
		t.Fatal("expected async plot analysis to be saved")
	}
}

func TestUpdateChapterPublishesChapterSavedEvent(t *testing.T) {
	bus := event.NewBus()
	var got ChapterSavedEvent
	bus.Subscribe(chapterSavedEventName, func(ctx context.Context, evt event.Event) error {
		got = evt.Payload.(ChapterSavedEvent)
		return nil
	})

	oldChapter := &data.Chapter{NovelID: 100, ChapterNo: 1, Title: "旧标题", Content: "旧内容", WordCount: 3}
	oldChapter.ID = 7
	uc := NewChapterUseCase(fakeNovelRepo{}, &fakeChapterRepo{find: oldChapter}, bus)

	if _, err := uc.UpdateChapter(context.Background(), dto.UpdateChapterRequest{
		NovelID:   100,
		ChapterID: 7,
		ChapterNo: 1,
		Title:     "新标题",
		Content:   "新内容",
		WordCount: 3,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.NovelID != 100 || got.ChapterID != 7 || got.Title != "新标题" || got.Content != "新内容" {
		t.Fatalf("unexpected event payload: %#v", got)
	}
}

func TestHandleChapterSavedUpsertsPlotAnalysis(t *testing.T) {
	repo := &fakeChapterPlotAnalysisRepo{}
	uc := &ChapterPlotAnalysisUseCase{
		analysisData: repo,
		analyzer: fakeChapterPlotAnalyzer{result: &ai_tools.ChapterPlotAnalysisTool{
			Summary:             "陆沉救下宋玉。",
			KeyEvents:           []string{"陆沉救下宋玉"},
			CharactersInvolved:  []string{"陆沉", "宋玉"},
			RelationshipChanges: []ai_tools.ChapterRelationshipChangeTool{{Source: "陆沉", Target: "宋玉", Change: "初次建立联系"}},
			EventAnalysis:       []string{"完成主角相遇"},
			Foreshadowing:       []string{"玉佩身份伏笔"},
			UnresolvedThreads:   []string{"追兵来源"},
		}},
	}

	err := uc.HandleChapterSaved(context.Background(), event.New(chapterSavedEventName, ChapterSavedEvent{
		NovelID:   100,
		ChapterID: 7,
		Title:     "第一章",
		Content:   "雨夜相逢。",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.saved == nil {
		t.Fatal("expected analysis to be upserted")
	}
	if repo.saved.NovelID != 100 || repo.saved.ChapterID != 7 || repo.saved.Summary != "陆沉救下宋玉。" {
		t.Fatalf("unexpected saved analysis: %#v", repo.saved)
	}
	if got := string(repo.saved.KeyEvents); got != `["陆沉救下宋玉"]` {
		t.Fatalf("unexpected key events json: %s", got)
	}
}

func TestGetChapterPlotAnalysisReturnsRepoError(t *testing.T) {
	uc := &ChapterPlotAnalysisUseCase{
		analysisData: &fakeChapterPlotAnalysisRepo{findErr: common.ChapterPlotAnalysisNotFound},
	}

	_, err := uc.GetChapterPlotAnalysis(context.Background(), 100, 7)
	if !errors.Is(err, common.ChapterPlotAnalysisNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestBuildChapterPlotAnalysisMessages(t *testing.T) {
	messages := buildChapterPlotAnalysisMessages("第一章 雨夜", "陆沉在雨夜救下宋玉。")
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	systemText := messages[0].ContentBlocks[0].UserInputText.Text
	if !strings.Contains(systemText, "chapter_plot_analysis_tool") {
		t.Fatalf("expected system prompt to require chapter plot analysis tool, got %q", systemText)
	}

	userText := messages[1].ContentBlocks[0].UserInputText.Text
	if !strings.Contains(userText, "第一章 雨夜") || !strings.Contains(userText, "陆沉在雨夜救下宋玉。") {
		t.Fatalf("unexpected user message: %q", userText)
	}
}

type fakeNovelRepo struct{}

func (fakeNovelRepo) Create(ctx context.Context, novel *data.Novel) (*data.Novel, error) {
	return novel, nil
}

func (fakeNovelRepo) List(ctx context.Context, offset, limit int) ([]data.Novel, int64, error) {
	return nil, 0, nil
}

func (fakeNovelRepo) FindByID(ctx context.Context, id uint) (*data.Novel, error) {
	return &data.Novel{}, nil
}

func (fakeNovelRepo) Update(ctx context.Context, novel *data.Novel) (*data.Novel, error) {
	return novel, nil
}

func (fakeNovelRepo) Delete(ctx context.Context, id uint) error {
	return nil
}

type fakeChapterRepo struct {
	find *data.Chapter
}

func (repo *fakeChapterRepo) Create(ctx context.Context, chapter *data.Chapter, wordDelta int64) (*data.Chapter, error) {
	chapter.ID = 7
	return chapter, nil
}

func (repo *fakeChapterRepo) List(ctx context.Context, novelID int64, offset, limit int) ([]data.Chapter, int64, error) {
	return nil, 0, nil
}

func (repo *fakeChapterRepo) FindByID(ctx context.Context, novelID int64, chapterID uint) (*data.Chapter, error) {
	if repo.find != nil {
		return repo.find, nil
	}
	return &data.Chapter{NovelID: novelID, ChapterNo: 1, Title: "旧标题", Content: "旧内容", WordCount: 3}, nil
}

func (repo *fakeChapterRepo) ChapterNoExists(ctx context.Context, novelID int64, chapterNo int, excludeID uint) (bool, error) {
	return false, nil
}

func (repo *fakeChapterRepo) MaxChapterNo(ctx context.Context, novelID int64) (int, error) {
	return 1, nil
}

func (repo *fakeChapterRepo) Update(ctx context.Context, chapter *data.Chapter, wordDelta int64) (*data.Chapter, error) {
	return chapter, nil
}

func (repo *fakeChapterRepo) Delete(ctx context.Context, novelID int64, chapterID uint, wordDelta int64) error {
	return nil
}

type fakeChapterPlotAnalyzer struct {
	result  *ai_tools.ChapterPlotAnalysisTool
	err     error
	started chan struct{}
	release chan struct{}
}

func (analyzer fakeChapterPlotAnalyzer) Analyze(ctx context.Context, title, content string) (*ai_tools.ChapterPlotAnalysisTool, error) {
	if analyzer.started != nil {
		close(analyzer.started)
	}
	if analyzer.release != nil {
		<-analyzer.release
	}

	return analyzer.result, analyzer.err
}

type fakeChapterPlotAnalysisRepo struct {
	saved   *data.ChapterPlotAnalysis
	savedCh chan struct{}
	find    *data.ChapterPlotAnalysis
	findErr error
}

func (repo *fakeChapterPlotAnalysisRepo) UpsertByChapterID(ctx context.Context, analysis *data.ChapterPlotAnalysis) (*data.ChapterPlotAnalysis, error) {
	repo.saved = analysis
	if repo.savedCh != nil {
		close(repo.savedCh)
		repo.savedCh = nil
	}
	return analysis, nil
}

func (repo *fakeChapterPlotAnalysisRepo) FindByChapterID(ctx context.Context, novelID int64, chapterID uint) (*data.ChapterPlotAnalysis, error) {
	if repo.findErr != nil {
		return nil, repo.findErr
	}
	return repo.find, nil
}
