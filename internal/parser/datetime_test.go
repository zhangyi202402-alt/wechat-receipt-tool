package parser

import (
	"testing"

	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

func TestPreferTwoDigitHourFromTimeColumn(t *testing.T) {
	blk := block{lines: []visualLine{
		{text: "二维码收款-来自陕AF39191..", topY: 192, bottom: 210, score: 0.94},
		{text: "+70.00", topY: 192, bottom: 210, score: 0.99},
		{text: "7月14日1:26", topY: 214, bottom: 228, score: 0.93},
	}}
	boxes := []ocr.TextBox{
		{Text: "7月14日1:26", Box: boxAt(60, 214, 200, 228), Score: 0.93},
		{Text: "7月14日16:26", Box: boxAt(60, 214, 200, 228), Score: 0.95, TimeColumn: true},
	}
	p, ok := bestDateTime("二维码收款-来自陕AF39191..\n+70.00\n7月14日1:26", blk, boxes)
	if !ok {
		t.Fatal("expected datetime")
	}
	if p.hour != 16 || p.minute != 26 {
		t.Fatalf("got %02d:%02d want 16:26", p.hour, p.minute)
	}
	if !p.fromTimeCol {
		t.Fatal("expected time column preference")
	}
}

func TestSingleDigitHourFlag(t *testing.T) {
	p := dateTimeParts{month: 7, day: 13, hour: 9, minute: 30, hourDigits: 1}
	_, tm, single := formatDateTime(p, 2026, 2026)
	if tm != "09:30" {
		t.Fatalf("time: %s", tm)
	}
	if !single {
		t.Fatal("expected single digit hour flag")
	}
}
