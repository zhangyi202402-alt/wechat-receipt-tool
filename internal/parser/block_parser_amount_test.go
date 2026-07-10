package parser

import "testing"

// 模拟 OCR 将 +300.00 拆碎时的解析结果
func TestBlockParser_AmountSplitVariants(t *testing.T) {
	cases := []struct {
		name   string
		lines  []FixtureLine
		amount float64
		ok     bool
	}{
		{
			name: "完整金额单框",
			lines: []FixtureLine{
				{Text: "二维码收款-来自*辛", Box: boxAt(60, 380, 280, 405), Score: 0.99},
				{Text: "+300.00", Box: boxAt(350, 380, 430, 405), Score: 0.99},
				{Text: "7月9日19:47", Box: boxAt(60, 412, 200, 432), Score: 0.97},
			},
			amount: 300,
			ok:     true,
		},
		{
			name: "OCR只认出+30",
			lines: []FixtureLine{
				{Text: "二维码收款-来自*辛", Box: boxAt(60, 380, 280, 405), Score: 0.99},
				{Text: "+30", Box: boxAt(350, 380, 430, 405), Score: 0.85},
				{Text: "7月9日19:47", Box: boxAt(60, 412, 200, 432), Score: 0.97},
			},
			amount: 30,
			ok:     true,
		},
		{
			name: "同行拆成+30与0.00",
			lines: []FixtureLine{
				{Text: "二维码收款-来自*辛", Box: boxAt(60, 380, 280, 405), Score: 0.99},
				{Text: "+30", Box: boxAt(350, 380, 390, 405), Score: 0.85},
				{Text: "0.00", Box: boxAt(392, 380, 430, 405), Score: 0.80},
				{Text: "7月9日19:47", Box: boxAt(60, 412, 200, 432), Score: 0.97},
			},
			amount: 30,
			ok:     true,
		},
		{
			name: "拆成+3与00.00",
			lines: []FixtureLine{
				{Text: "二维码收款-来自*辛", Box: boxAt(60, 380, 280, 405), Score: 0.99},
				{Text: "+3", Box: boxAt(350, 380, 370, 405), Score: 0.85},
				{Text: "00.00", Box: boxAt(372, 380, 430, 405), Score: 0.80},
				{Text: "7月9日19:47", Box: boxAt(60, 412, 200, 432), Score: 0.97},
			},
			amount: 3,
			ok:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recs := ParseFromLines(tc.lines, Options{FallbackYear: 2026, StoreName: "测试"})
			if !tc.ok {
				if len(recs) != 0 {
					t.Fatalf("expected no record, got %d", len(recs))
				}
				return
			}
			if len(recs) != 1 {
				t.Fatalf("expected 1 record, got %d", len(recs))
			}
			if recs[0].Amount != tc.amount {
				t.Errorf("amount: got %.2f want %.2f (combined block would use regex on joined text)", recs[0].Amount, tc.amount)
			}
		})
	}
}

func boxAt(x1, y1, x2, y2 float64) [4][2]float64 {
	return [4][2]float64{{x1, y1}, {x2, y1}, {x2, y2}, {x1, y2}}
}
