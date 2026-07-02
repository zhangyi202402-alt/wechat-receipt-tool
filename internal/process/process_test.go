package process

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
	"github.com/kalading/wechat-receipt-tool/internal/parser"
)

type fixtureOCREngine struct {
	lines []parser.FixtureLine
}

func (e *fixtureOCREngine) Init(_ string, _ config.OCRConfig) error { return nil }
func (e *fixtureOCREngine) Close() error                             { return nil }

func (e *fixtureOCREngine) Recognize(_ string) ([]ocr.TextBox, error) {
	var boxes []ocr.TextBox
	for _, ln := range e.lines {
		boxes = append(boxes, ocr.TextBox{Text: ln.Text, Box: ln.Box, Score: ln.Score})
	}
	return boxes, nil
}

func loadFixtureLines(t *testing.T) []parser.FixtureLine {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "blocks_fixture.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []parser.FixtureLine
	if err := json.Unmarshal(data, &lines); err != nil {
		t.Fatal(err)
	}
	return lines
}

func TestService_MergedExcelAtDateDir(t *testing.T) {
	dir := t.TempDir()
	store := "北京世纪金源店"
	dateStr := "2026-07-02"
	storeDir := filepath.Join(dir, dateStr, store)
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "fake.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		BaseDir:         ".",
		DateFormat:      "2006-01-02",
		Stores:          []string{store},
		OutputFilename:  "收款记录.xlsx",
		ReportFilename:  "处理报告.txt",
		ImageExtensions: []string{".png"},
		OCR:             config.OCRConfig{Workers: 1},
	}

	svc := NewService(cfg, dir, &fixtureOCREngine{lines: loadFixtureLines(t)})
	date, _ := time.Parse("2006-01-02", dateStr)
	result, err := svc.Run(date, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalDedup != 3 {
		t.Fatalf("total dedup: %d", result.TotalDedup)
	}
	if !result.ExcelWritten {
		t.Fatal("excel not written")
	}
	dateExcel := filepath.Join(dir, dateStr, "收款记录.xlsx")
	dateReport := filepath.Join(dir, dateStr, "处理报告.txt")
	if _, err := os.Stat(dateExcel); err != nil {
		t.Fatalf("merged excel missing: %v", err)
	}
	if _, err := os.Stat(dateReport); err != nil {
		t.Fatalf("merged report missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(storeDir, "收款记录.xlsx")); err == nil {
		t.Fatal("excel should not be in store subdir")
	}
}
