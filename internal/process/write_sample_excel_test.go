//go:build e2e

package process

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

func TestWriteSampleExcel_BillJul14(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	work := filepath.Join(root, "testdata", "e2e_embed_out")
	store := "北京世纪金源店"
	trueVal := true
	ort := filepath.Join(root, "release", "lib", "libonnxruntime.dylib")
	models := filepath.Join(root, "release", "models")
	cfg := &config.Config{
		BaseDir: ".", DateFormat: "2006-01-02", Stores: []string{store},
		OutputFilename: "收款记录.xlsx", ReportFilename: "处理报告.txt",
		ImageExtensions: []string{".png"},
		OCR: config.OCRConfig{
			Provider: "gopaddleocr", OnnxRuntimeLib: ort, ModelsDir: models, Workers: 1,
			AmountColumnOCR: &trueVal, AmountColumnStart: 0.75, AmountColumnScale: 2,
			TimeColumnOCR: &trueVal, TimeColumnEnd: 0.55, TimeColumnScale: 2,
		},
		Process: config.ProcessConfig{
			Overwrite: true, IncludeTypes: []string{"all"}, RequireDate: false,
			ReviewConfidenceBelow: 0.85, SaveReviewSnippets: &trueVal, EmbedReviewImages: &trueVal,
		},
	}
	engine, err := ocr.NewEngine(cfg.OCR.Provider)
	if err != nil { t.Fatal(err) }
	if err := engine.Init(work, cfg.OCR); err != nil { t.Fatal(err) }
	defer engine.Close()
	svc := NewService(cfg, work, engine)
	date, _ := time.Parse("2006-01-02", "2026-07-14")
	res, err := svc.Run(date, store, true, false)
	if err != nil { t.Fatal(err) }
	t.Log("Excel written to:", res.ExcelPath)
	if _, err := os.Stat(res.ExcelPath); err != nil { t.Fatal(err) }
}
