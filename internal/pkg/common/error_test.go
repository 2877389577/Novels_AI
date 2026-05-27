package common

import (
	"net/http"
	"testing"
)

func TestChapterPlotAnalysisNotFoundUsesHTTPStatusOK(t *testing.T) {
	// 章节剧情总结由异步任务生成，刚保存章节后查不到总结属于正常业务状态，不能让 HTTP 层返回 404。
	if ChapterPlotAnalysisNotFound.StatusCode() != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", ChapterPlotAnalysisNotFound.StatusCode())
	}
	if ChapterPlotAnalysisNotFound.Code != 3008 {
		t.Fatalf("expected business code 3008, got %d", ChapterPlotAnalysisNotFound.Code)
	}
}
