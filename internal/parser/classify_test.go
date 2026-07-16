package parser

import (
	"testing"

	"github.com/kalading/wechat-receipt-tool/internal/models"
)

func TestClassifyAndMixedTypes(t *testing.T) {
	lines := []FixtureLine{
		{Text: "微信红包-通过企业微信领取", Box: boxAt(40, 100, 320, 120), Score: 0.95},
		{Text: "+115.00", Box: boxAt(500, 100, 600, 120), Score: 0.95},
		{Text: "7月14日 10:01", Box: boxAt(40, 125, 200, 140), Score: 0.95},

		{Text: "转账-转给雷尔", Box: boxAt(40, 180, 280, 200), Score: 0.95},
		{Text: "-3900.00", Box: boxAt(500, 180, 620, 200), Score: 0.95},
		{Text: "7月14日 16:26", Box: boxAt(40, 205, 200, 220), Score: 0.95},

		{Text: "转账-来自雷尔", Box: boxAt(40, 260, 280, 280), Score: 0.95},
		{Text: "+3900.00", Box: boxAt(500, 260, 620, 280), Score: 0.95},
		{Text: "7月14日 16:30", Box: boxAt(40, 285, 200, 300), Score: 0.95},

		{Text: "二维码收款-来自*颜", Box: boxAt(40, 340, 300, 360), Score: 0.95},
		{Text: "+70.00", Box: boxAt(500, 340, 580, 360), Score: 0.95},
		{Text: "7月14日 18:00", Box: boxAt(40, 365, 200, 380), Score: 0.95},

		{Text: "卓越车匠维", Box: boxAt(40, 420, 220, 440), Score: 0.92},
		{Text: "-300.00", Box: boxAt(500, 420, 600, 440), Score: 0.95},
		{Text: "7月14日 19:10", Box: boxAt(40, 445, 200, 460), Score: 0.95},
	}
	recs := Deduplicate(ParseFromLines(lines, Options{FallbackYear: 2026, StoreName: "测试店"}))
	if len(recs) != 5 {
		t.Fatalf("expected 5 records, got %d", len(recs))
	}

	byType := map[models.TxType]int{}
	for _, r := range recs {
		byType[r.Type]++
		if r.Amount == 0 {
			t.Errorf("zero amount for %s", r.Source)
		}
	}
	if byType[models.TxRedPacket] != 1 {
		t.Errorf("red packet count: %d", byType[models.TxRedPacket])
	}
	if byType[models.TxTransferOut] != 1 {
		t.Errorf("transfer out: %d", byType[models.TxTransferOut])
	}
	if byType[models.TxTransferIn] != 1 {
		t.Errorf("transfer in: %d", byType[models.TxTransferIn])
	}
	if byType[models.TxQRReceipt] != 1 {
		t.Errorf("qr: %d", byType[models.TxQRReceipt])
	}
	if byType[models.TxMerchant] != 1 {
		t.Errorf("merchant: %d", byType[models.TxMerchant])
	}

	var outAmt, inAmt float64
	for _, r := range recs {
		if r.Type == models.TxTransferOut {
			outAmt = r.Amount
			if r.Direction != models.DirectionOut {
				t.Errorf("transfer out direction: %s", r.Direction)
			}
			if r.Source != "雷尔" {
				t.Errorf("transfer out source: %q", r.Source)
			}
		}
		if r.Type == models.TxTransferIn {
			inAmt = r.Amount
		}
	}
	if outAmt != -3900 {
		t.Errorf("transfer out amount: %.2f", outAmt)
	}
	if inAmt != 3900 {
		t.Errorf("transfer in amount: %.2f", inAmt)
	}
}

func TestPartialRecordMissingDate(t *testing.T) {
	lines := []FixtureLine{
		{Text: "二维码收款-来自*颜", Box: boxAt(40, 100, 300, 120), Score: 0.95},
		{Text: "+70.00", Box: boxAt(500, 100, 580, 120), Score: 0.95},
	}
	recs := ParseFromLines(lines, Options{FallbackYear: 2026, StoreName: "测试"})
	if len(recs) != 1 {
		t.Fatalf("expected 1 record without date, got %d", len(recs))
	}
	if recs[0].Status != models.StatusNeedsReview {
		t.Errorf("status: %s", recs[0].Status)
	}
	found := false
	for _, r := range recs[0].ReviewReasons {
		if r == "缺日期" {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons: %v", recs[0].ReviewReasons)
	}
}

func TestRequireDateDropsIncomplete(t *testing.T) {
	lines := []FixtureLine{
		{Text: "二维码收款-来自*颜", Box: boxAt(40, 100, 300, 120), Score: 0.95},
		{Text: "+70.00", Box: boxAt(500, 100, 580, 120), Score: 0.95},
	}
	recs := ParseFromLines(lines, Options{FallbackYear: 2026, RequireDate: true})
	if len(recs) != 0 {
		t.Fatalf("expected 0 with require_date, got %d", len(recs))
	}
}

func TestDeduplicateKeepsDifferentTypes(t *testing.T) {
	records := []models.ReceiptRecord{
		{Date: "2026-07-14", Time: "16:26", Amount: 3900, Type: models.TxTransferIn, OCRScore: 0.9},
		{Date: "2026-07-14", Time: "16:26", Amount: -3900, Type: models.TxTransferOut, OCRScore: 0.9},
	}
	out := Deduplicate(records)
	if len(out) != 2 {
		t.Fatalf("expected 2 different types, got %d", len(out))
	}
}
