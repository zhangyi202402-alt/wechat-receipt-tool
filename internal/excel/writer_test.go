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
		{Serial: 1, Transferor: "北京世纪金源店", Source: "*李", Amount: 1134, Date: "2026-07-01", Time: "15:48"},
		{Serial: 2, Transferor: "北京世纪金源店", Source: "孔宪光", Amount: 450, Date: "2026-06-24", Time: "08:20"},
	}
	if err := NewWriter().Write(path, records); err != nil {
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
	if v, _ := f.GetCellValue("Sheet1", "C3"); v != "*李" {
		t.Errorf("C3: got %q", v)
	}
	if v, _ := f.GetCellValue("Sheet1", "A2"); v == "" {
		t.Error("note row missing")
	}
	if v, _ := f.GetCellValue("Sheet1", "A5"); v != "合计" {
		t.Errorf("total label: got %q", v)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
