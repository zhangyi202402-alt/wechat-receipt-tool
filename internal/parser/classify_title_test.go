package parser

import (
	"strings"
	"testing"

	"github.com/kalading/wechat-receipt-tool/internal/models"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

// bill_jul14 风格：整图完整标题 + 时间列碎片，应取完整对方
func TestClassify_PreferFullTitleOverTimeFragment(t *testing.T) {
	boxes := []ocr.TextBox{
		{Text: "二维码收款-来自*创", Box: boxAt(60, 360, 280, 380), Score: 0.99},
		{Text: "二维码收款-来自*t", Box: boxAt(50, 360, 200, 380), Score: 0.93, TimeColumn: true},
		{Text: "+58.00", Box: boxAt(400, 360, 480, 380), Score: 0.99},
		{Text: "+58.00", Box: boxAt(400, 360, 480, 380), Score: 0.96, AmountColumn: true},
		{Text: "7月14日15:27", Box: boxAt(60, 390, 200, 410), Score: 0.97},
		{Text: "7月14日15:27", Box: boxAt(60, 390, 200, 410), Score: 0.98, TimeColumn: true},
	}
	recs := NewBlockParser(Options{FallbackYear: 2026, StoreName: "测试"}).Parse(boxes)
	if len(recs) != 1 {
		t.Fatalf("expected 1, got %d", len(recs))
	}
	if recs[0].Source != "*创" {
		t.Errorf("source: got %q want *创", recs[0].Source)
	}
	if recs[0].Amount != 58 {
		t.Errorf("amount: %.2f", recs[0].Amount)
	}
}

func TestClassify_LongestQRSource(t *testing.T) {
	lines := []FixtureLine{
		{Text: "二维码收款-来自陕AF39191..", Box: boxAt(60, 270, 320, 290), Score: 0.95},
		{Text: "+70.00", Box: boxAt(400, 270, 480, 290), Score: 0.99},
		{Text: "7月14日 16:26", Box: boxAt(60, 300, 200, 320), Score: 0.94},
	}
	recs := ParseFromLines(lines, Options{FallbackYear: 2026})
	if len(recs) != 1 {
		t.Fatalf("expected 1, got %d", len(recs))
	}
	if !strings.HasPrefix(recs[0].Source, "陕AF39191") {
		t.Errorf("source: got %q", recs[0].Source)
	}
}

func TestClassify_MerchantTitleClean(t *testing.T) {
	lines := []FixtureLine{
		{Text: "西安市高新区卓越车匠维…", Box: boxAt(60, 900, 320, 920), Score: 0.96},
		{Text: "-300.00", Box: boxAt(400, 900, 480, 920), Score: 0.99},
		{Text: "7月11日18:23", Box: boxAt(60, 930, 200, 950), Score: 0.95},
	}
	recs := ParseFromLines(lines, Options{FallbackYear: 2026})
	if len(recs) != 1 {
		t.Fatalf("expected 1, got %d", len(recs))
	}
	if recs[0].Type != models.TxMerchant {
		t.Errorf("type: %s", recs[0].Type)
	}
	if strings.Contains(recs[0].Source, "7月") {
		t.Errorf("merchant source should not contain date: %q", recs[0].Source)
	}
	if !strings.Contains(recs[0].Source, "卓越车匠维") {
		t.Errorf("source: %q", recs[0].Source)
	}
	if recs[0].Amount != -300 {
		t.Errorf("amount: %.2f", recs[0].Amount)
	}
}

func TestMergeAmount_PreferFullWhenLarger(t *testing.T) {
	amt, ok, conflict := mergeAmountChoice(100, 10, true, true)
	if !ok || amt != 100 {
		t.Fatalf("got %.2f ok=%v", amt, ok)
	}
	if !conflict {
		t.Error("expected conflict flag")
	}
}

func TestBillJul14_GoldenLike(t *testing.T) {
	// 模拟 bill_jul14 关键 9 条（整图为主，不含时间列碎片）
	lines := []FixtureLine{
		{Text: "2026年7月", Box: boxAt(20, 200, 120, 220), Score: 1},
		{Text: "二维码收款-来自陕AF39191..", Box: boxAt(60, 270, 320, 290), Score: 0.95},
		{Text: "+70.00", Box: boxAt(400, 270, 480, 290), Score: 0.99},
		{Text: "7月14日16:26", Box: boxAt(60, 300, 200, 320), Score: 0.94},

		{Text: "二维码收款-来自*创", Box: boxAt(60, 360, 280, 380), Score: 0.99},
		{Text: "+58.00", Box: boxAt(400, 360, 480, 380), Score: 0.99},
		{Text: "7月14日15:27", Box: boxAt(60, 390, 200, 410), Score: 0.97},

		{Text: "二维码收款-来自孟*2", Box: boxAt(60, 450, 280, 470), Score: 0.99},
		{Text: "+500.00", Box: boxAt(400, 450, 500, 470), Score: 0.99},
		{Text: "7月14日12:39", Box: boxAt(60, 480, 200, 500), Score: 0.97},

		{Text: "二维码收款-来自L*n", Box: boxAt(60, 540, 280, 560), Score: 0.99},
		{Text: "+1448.00", Box: boxAt(400, 540, 520, 560), Score: 0.99},
		{Text: "7月13日12:59", Box: boxAt(60, 570, 200, 590), Score: 0.96},

		{Text: "二维码收款-来自*庐", Box: boxAt(60, 630, 280, 650), Score: 0.99},
		{Text: "+444.00", Box: boxAt(400, 630, 500, 650), Score: 0.99},
		{Text: "7月13日11:50", Box: boxAt(60, 660, 200, 680), Score: 0.97},

		{Text: "二维码收款-来自*迹", Box: boxAt(60, 720, 280, 740), Score: 0.99},
		{Text: "+100.00", Box: boxAt(400, 720, 500, 740), Score: 0.99},
		{Text: "7月13日11:44", Box: boxAt(60, 750, 200, 770), Score: 0.98},

		{Text: "二维码收款-来自*海", Box: boxAt(60, 810, 280, 830), Score: 0.99},
		{Text: "+4800.00", Box: boxAt(400, 810, 520, 830), Score: 0.99},
		{Text: "7月13日09:30", Box: boxAt(60, 840, 200, 860), Score: 0.97},

		{Text: "西安市高新区卓越车匠维…", Box: boxAt(60, 900, 340, 920), Score: 0.96},
		{Text: "-300.00", Box: boxAt(400, 900, 500, 920), Score: 1.0},
		{Text: "7月11日18:23", Box: boxAt(60, 930, 200, 950), Score: 0.95},

		{Text: "二维码收款-来自*颜", Box: boxAt(60, 1000, 280, 1020), Score: 0.99},
		{Text: "+300.00", Box: boxAt(400, 1000, 500, 1020), Score: 0.99},
	}
	recs := Deduplicate(ParseFromLines(lines, Options{FallbackYear: 2026, StoreName: "测试店"}))
	if len(recs) != 9 {
		t.Fatalf("expected 9 records, got %d", len(recs))
		for _, r := range recs {
			t.Logf("  %s %s %.2f %s", r.Type, r.Source, r.Amount, r.Date)
		}
	}

	want := []struct {
		srcPrefix string
		amount    float64
		date      string
		txType    models.TxType
	}{
		{"陕AF39191", 70, "2026-07-14", models.TxQRReceipt},
		{"*创", 58, "2026-07-14", models.TxQRReceipt},
		{"孟*2", 500, "2026-07-14", models.TxQRReceipt},
		{"L*n", 1448, "2026-07-13", models.TxQRReceipt},
		{"*庐", 444, "2026-07-13", models.TxQRReceipt},
		{"*迹", 100, "2026-07-13", models.TxQRReceipt},
		{"*海", 4800, "2026-07-13", models.TxQRReceipt},
		{"卓越车匠维", -300, "2026-07-11", models.TxMerchant},
		{"*颜", 300, "", models.TxQRReceipt},
	}

	for _, w := range want {
		found := false
		for _, r := range recs {
			if r.Amount != w.amount || r.Type != w.txType {
				continue
			}
			if !strings.Contains(r.Source, w.srcPrefix) {
				continue
			}
			if w.date != "" && r.Date != w.date {
				continue
			}
			if w.date == "" && r.Date != "" {
				continue
			}
			found = true
			if w.date == "" && r.Status != models.StatusNeedsReview {
				t.Errorf("%s: expected needs_review, got %s", w.srcPrefix, r.Status)
			}
			break
		}
		if !found {
			t.Errorf("missing record: type=%s src~%s amount=%.0f date=%s", w.txType, w.srcPrefix, w.amount, w.date)
		}
	}
}
