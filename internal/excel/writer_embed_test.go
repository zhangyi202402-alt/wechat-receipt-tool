package excel

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/kalading/wechat-receipt-tool/internal/models"
)

func TestWriter_EmbedsReviewSnippet(t *testing.T) {
	dir := t.TempDir()
	snippetRel := filepath.ToSlash(filepath.Join("review", "测试店", "001_test.png"))
	snippetAbs := filepath.Join(dir, filepath.FromSlash(snippetRel))
	if err := os.MkdirAll(filepath.Dir(snippetAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 120, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 200, B: 0, A: 255})
		}
	}
	f, err := os.Create(snippetAbs)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	path := filepath.Join(dir, "out.xlsx")
	records := []models.ReceiptRecord{
		{
			Serial: 1, Transferor: "测试店", Source: "*颜", Amount: 300,
			Type: models.TxQRReceipt, Direction: models.DirectionIn,
			Status: models.StatusNeedsReview, Confidence: 0.7,
			ReviewReasons:  []string{"缺日期"},
			SnippetRelPath: snippetRel,
		},
	}
	if err := NewWriter().Write(path, records, WriteOptions{
		EmbedReviewImages: true,
		DateDir:           dir,
	}); err != nil {
		t.Fatal(err)
	}

	xf, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer xf.Close()

	pics, err := xf.GetPictures("Sheet1", "L3")
	if err != nil {
		t.Fatal(err)
	}
	if len(pics) == 0 {
		t.Fatal("expected embedded picture in L3 for needs_review row")
	}
	// 不应再有「片段路径」列
	if v, _ := xf.GetCellValue("Sheet1", "L1"); v != "原图片段" {
		t.Errorf("L1 header: got %q", v)
	}
	if v, _ := xf.GetCellValue("Sheet1", "M1"); v != "" {
		t.Errorf("unexpected M1 column: %q", v)
	}
}

func TestWriter_EmbedMissingImageNoCrash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.xlsx")
	records := []models.ReceiptRecord{
		{
			Serial: 1, Transferor: "测试店", Source: "*颜", Amount: 300,
			Type: models.TxQRReceipt, Status: models.StatusNeedsReview,
			Confidence: 0.7, ReviewReasons: []string{"缺日期"},
			SnippetRelPath: "review/测试店/missing.png",
		},
	}
	if err := NewWriter().Write(path, records, WriteOptions{
		EmbedReviewImages: true,
		DateDir:           dir,
	}); err != nil {
		t.Fatal(err)
	}
}
