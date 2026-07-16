package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/kalading/wechat-receipt-tool/internal/models"
)

func TestWriter_Output(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.xlsx")
	records := []models.ReceiptRecord{
		{
			Serial: 1, Transferor: "北京世纪金源店", Source: "*李", Amount: 1134,
			Date: "2026-07-01", Time: "15:48", Type: models.TxQRReceipt,
			Direction: models.DirectionIn, Status: models.StatusOK, Confidence: 0.95,
		},
		{
			Serial: 2, Transferor: "北京世纪金源店", Source: "雷尔", Amount: -3900,
			Date: "2026-07-14", Time: "16:26", Type: models.TxTransferOut,
			Direction: models.DirectionOut, Status: models.StatusNeedsReview,
			Confidence: 0.7, ReviewReasons: []string{"缺日期"},
		},
	}
	if err := NewWriter().Write(path, records, WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if v, _ := f.GetCellValue("Sheet1", "B3"); v != "北京世纪金源店" {
		t.Errorf("B3: got %q", v)
	}
	if v, _ := f.GetCellValue("Sheet1", "C3"); v != "二维码收款" {
		t.Errorf("C3 type: got %q", v)
	}
	if v, _ := f.GetCellValue("Sheet1", "E3"); v != "*李" {
		t.Errorf("E3 source: got %q", v)
	}
	if v, _ := f.GetCellValue("Sheet1", "K4"); v != "缺日期" {
		t.Errorf("K4 reason: got %q", v)
	}
	if v, _ := f.GetCellValue("Sheet1", "A2"); v == "" {
		t.Error("note row missing")
	}
	if v, _ := f.GetCellValue("Sheet1", "A5"); v != "合计(净额)" {
		t.Errorf("total label: got %q", v)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
