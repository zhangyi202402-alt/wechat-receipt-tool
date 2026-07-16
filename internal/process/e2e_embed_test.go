//go:build e2e

package process

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

func TestE2E_BillJul14_EmbedReviewImage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mac e2e")
	}
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	imgSrc := filepath.Join(root, "testdata", "bill_jul14.png")
	if _, err := os.Stat(imgSrc); err != nil {
		t.Skip("bill_jul14.png missing")
	}
	ort := filepath.Join(root, "release", "lib", "libonnxruntime.dylib")
	models := filepath.Join(root, "release", "models")
	if _, err := os.Stat(ort); err != nil {
		t.Skip("onnxruntime not available")
	}

	work := t.TempDir()
	store := "北京世纪金源店"
	dateStr := "2026-07-14"
	storeDir := filepath.Join(work, dateStr, store)
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(imgSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "bill_jul14.png"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	trueVal := true
	cfg := &config.Config{
		BaseDir:         ".",
		DateFormat:      "2006-01-02",
		Stores:          []string{store},
		OutputFilename:  "收款记录.xlsx",
		ReportFilename:  "处理报告.txt",
		ImageExtensions: []string{".png", ".jpg"},
		OCR: config.OCRConfig{
			Provider:          "gopaddleocr",
			OnnxRuntimeLib:    ort,
			ModelsDir:         models,
			Workers:           1,
			AmountColumnOCR:   &trueVal,
			AmountColumnStart: 0.75,
			AmountColumnScale: 2.0,
			TimeColumnOCR:     &trueVal,
			TimeColumnEnd:     0.55,
			TimeColumnScale:   2.0,
		},
		Process: config.ProcessConfig{
			Overwrite:             true,
			IncludeTypes:          []string{"all"},
			RequireDate:           false,
			ReviewConfidenceBelow: 0.85,
			SaveReviewSnippets:    &trueVal,
			EmbedReviewImages:     &trueVal,
		},
	}

	engine, err := ocr.NewEngine(cfg.OCR.Provider)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Init(work, cfg.OCR); err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	svc := NewService(cfg, work, engine)
	date, _ := time.Parse("2006-01-02", dateStr)
	res, err := svc.Run(date, store, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.ExcelWritten {
		t.Fatal("excel not written")
	}
	t.Logf("dedup=%d excel=%s", res.TotalDedup, res.ExcelPath)

	f, err := excelize.OpenFile(res.ExcelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if h, _ := f.GetCellValue("Sheet1", "L1"); h != "原图片段" {
		t.Fatalf("L1=%q", h)
	}
	if h, _ := f.GetCellValue("Sheet1", "M1"); h != "" {
		t.Fatalf("unexpected path column M1=%q", h)
	}

	reviewRows, embedded := 0, 0
	for row := 3; row < 40; row++ {
		status, _ := f.GetCellValue("Sheet1", fmt.Sprintf("I%d", row))
		if status == "" || status == "合计(净额)" {
			break
		}
		src, _ := f.GetCellValue("Sheet1", fmt.Sprintf("E%d", row))
		amt, _ := f.GetCellValue("Sheet1", fmt.Sprintf("F%d", row))
		t.Logf("row%d %s src=%s amt=%s", row, status, src, amt)
		if status != "待核对" {
			continue
		}
		reviewRows++
		pics, err := f.GetPictures("Sheet1", fmt.Sprintf("L%d", row))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("  L%d pictures=%d", row, len(pics))
		if len(pics) > 0 {
			embedded++
			out := filepath.Join(work, fmt.Sprintf("embedded_row%d.png", row))
			_ = os.WriteFile(out, pics[0].File, 0o644)
			t.Logf("  extracted %s (%d bytes)", out, len(pics[0].File))
		}
	}
	if reviewRows == 0 {
		t.Fatal("expected needs_review row (*颜)")
	}
	if embedded != reviewRows {
		t.Fatalf("embed mismatch review=%d embedded=%d", reviewRows, embedded)
	}
}
