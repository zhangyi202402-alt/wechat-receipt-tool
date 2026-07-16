package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kalading/wechat-receipt-tool/internal/models"
)

func loadFixture(t *testing.T) []FixtureLine {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "blocks_fixture.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []FixtureLine
	if err := json.Unmarshal(data, &lines); err != nil {
		t.Fatal(err)
	}
	return lines
}

func TestBlockParser_QRReceipts(t *testing.T) {
	lines := loadFixture(t)
	records := Deduplicate(ParseFromLines(lines, Options{
		FallbackYear: 2026,
		StoreName:    "北京世纪金源店",
		SourceImage:  "sample.png",
	}))

	if len(records) != 4 {
		t.Fatalf("expected 4 records (3 QR + 1 refund), got %d", len(records))
	}

	qrCount := 0
	var june *models.ReceiptRecord
	for i := range records {
		r := &records[i]
		if r.Type == models.TxQRReceipt {
			qrCount++
		}
		if r.Date == "2026-06-30" && r.Amount == 1134 {
			june = r
		}
	}
	if qrCount != 3 {
		t.Fatalf("expected 3 QR records, got %d", qrCount)
	}
	if june == nil {
		t.Fatal("expected June 30 record")
	}
	if june.Source != "*李" {
		t.Errorf("source: got %q", june.Source)
	}
	if june.Transferor != "北京世纪金源店" {
		t.Errorf("transferor: got %q", june.Transferor)
	}

	foundJuly := false
	foundRefund := false
	for _, r := range records {
		if r.Date == "2026-07-01" && r.Amount == 1 {
			foundJuly = true
		}
		if r.Type == models.TxRefund && r.Amount == 10 {
			foundRefund = true
		}
	}
	if !foundJuly {
		t.Error("expected July 1.00 record")
	}
	if !foundRefund {
		t.Error("expected refund +10 record")
	}
}

func TestDeduplicate(t *testing.T) {
	records := []models.ReceiptRecord{
		{Date: "2026-07-01", Time: "15:48", Amount: 1134, Source: "*李", Type: models.TxQRReceipt, OCRScore: 0.9},
		{Date: "2026-07-01", Time: "15:48", Amount: 1134, Source: "*李", Type: models.TxQRReceipt, OCRScore: 0.95},
	}
	out := Deduplicate(records)
	if len(out) != 1 {
		t.Fatalf("expected 1 after dedup, got %d", len(out))
	}
	if out[0].Serial != 1 {
		t.Errorf("serial: got %d", out[0].Serial)
	}
}

func TestBlockParser_FallbackYear(t *testing.T) {
	lines := []FixtureLine{
		{Text: "二维码收款-来自测试", Box: [4][2]float64{{60, 150}, {280, 150}, {280, 175}, {60, 175}}, Score: 0.9},
		{Text: "+100.00", Box: [4][2]float64{{600, 150}, {720, 150}, {720, 175}, {600, 175}}, Score: 0.9},
		{Text: "3月5日 10:30", Box: [4][2]float64{{60, 185}, {200, 185}, {200, 205}, {60, 205}}, Score: 0.9},
	}
	records := ParseFromLines(lines, Options{FallbackYear: 2025, StoreName: "测试店"})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Date != "2025-03-05" {
		t.Errorf("date: got %s", records[0].Date)
	}
}
