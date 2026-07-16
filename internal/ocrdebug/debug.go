package ocrdebug

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/kalading/wechat-receipt-tool/internal/models"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
	"github.com/kalading/wechat-receipt-tool/internal/parser"
)

// WriteImageSection dumps OCR boxes and parsed records for one screenshot.
func WriteImageSection(w io.Writer, store, imageName string, boxes []ocr.TextBox, records []models.ReceiptRecord) {
	fmt.Fprintf(w, "=== %s | 门店: %s ===\n", imageName, store)
	fmt.Fprintf(w, "OCR 原始框 (%d):\n", len(boxes))
	for i, line := range formatBoxes(boxes) {
		tag := " "
		if line.amountCol {
			tag = " [金额列]"
		} else if line.timeCol {
			tag = " [时间列]"
		}
		fmt.Fprintf(w, "  %3d y=%6.0f x=%6.0f score=%.3f%s %q\n",
			i+1, line.y, line.x, line.score, tag, line.text)
	}
	fmt.Fprintf(w, "解析记录 (%d):\n", len(records))
	if len(records) == 0 {
		fmt.Fprintln(w, "  (无)")
	} else {
		for _, r := range records {
			status := r.StatusLabelCN()
			reasons := ""
			if len(r.ReviewReasons) > 0 {
				reasons = " 原因=" + strings.Join(r.ReviewReasons, "；")
			}
			fmt.Fprintf(w, "  类型=%s 对方=%s 金额=%.2f 日期=%s %s 状态=%s 置信度=%.0f%%%s\n",
				r.Type.LabelCN(), r.Source, r.Amount, r.Date, r.Time, status, r.Confidence*100, reasons)
		}
	}
	fmt.Fprintln(w, "交易块预览:")
	for i, block := range previewTxBlocks(boxes) {
		fmt.Fprintf(w, "  block %d: %s\n", i+1, block)
	}
	fmt.Fprintln(w)
}

func formatBoxes(boxes []ocr.TextBox) []boxLine {
	sorted := append([]ocr.TextBox(nil), boxes...)
	sort.Slice(sorted, func(i, j int) bool {
		yi := centerY(sorted[i].Box)
		yj := centerY(sorted[j].Box)
		if math.Abs(yi-yj) < 1 {
			return centerX(sorted[i].Box) < centerX(sorted[j].Box)
		}
		return yi < yj
	})
	out := make([]boxLine, len(sorted))
	for i, b := range sorted {
		out[i] = boxLine{
			text:      b.Text,
			x:         centerX(b.Box),
			y:         centerY(b.Box),
			score:     b.Score,
			amountCol: b.AmountColumn,
			timeCol:   b.TimeColumn,
		}
	}
	return out
}

type boxLine struct {
	text      string
	x, y      float64
	score     float64
	amountCol bool
	timeCol   bool
}

func previewTxBlocks(boxes []ocr.TextBox) []string {
	recs := parser.ParseFromLines(boxesToFixture(boxes), parser.Options{FallbackYear: 2026})
	if len(recs) == 0 {
		// fallback: show raw anchors
		return previewAnchors(boxes)
	}
	var out []string
	for _, r := range recs {
		out = append(out, fmt.Sprintf("%s | %s | %.2f | %s %s | %s",
			r.Type.LabelCN(), r.Source, r.Amount, r.Date, r.Time, r.StatusLabelCN()))
	}
	return out
}

func boxesToFixture(boxes []ocr.TextBox) []parser.FixtureLine {
	out := make([]parser.FixtureLine, len(boxes))
	for i, b := range boxes {
		out[i] = parser.FixtureLine{Text: b.Text, Box: b.Box, Score: b.Score}
	}
	return out
}

func previewAnchors(boxes []ocr.TextBox) []string {
	sorted := append([]ocr.TextBox(nil), boxes...)
	sort.Slice(sorted, func(i, j int) bool {
		return centerY(sorted[i].Box) < centerY(sorted[j].Box)
	})
	var blocks []string
	for i, b := range sorted {
		text := b.Text
		isAmt := strings.Contains(text, "+") || strings.Contains(text, "-")
		isTitle := strings.Contains(text, "二维码收款") || strings.Contains(text, "转账") ||
			strings.Contains(text, "红包") || strings.Contains(text, "退款") || strings.Contains(text, "提现")
		if !isAmt && !isTitle {
			continue
		}
		startY := centerY(b.Box)
		var parts []string
		for j := i; j < len(sorted); j++ {
			cy := centerY(sorted[j].Box)
			if j > i && cy-startY > 120 {
				break
			}
			parts = append(parts, fmt.Sprintf("%q", sorted[j].Text))
			if j > i {
				tj := sorted[j].Text
				if strings.Contains(tj, "二维码收款") || strings.Contains(tj, "转账") ||
					strings.Contains(tj, "红包") {
					if j > i {
						break
					}
				}
			}
		}
		blocks = append(blocks, strings.Join(parts, " + "))
	}
	return blocks
}

func centerY(box [4][2]float64) float64 {
	return (box[0][1] + box[2][1]) / 2
}

func centerX(box [4][2]float64) float64 {
	return (box[0][0] + box[2][0]) / 2
}

// ParseRecords is a helper for standalone debug tools.
func ParseRecords(boxes []ocr.TextBox, fallbackYear int, store, imageName string) []models.ReceiptRecord {
	return parser.NewBlockParser(parser.Options{
		FallbackYear: fallbackYear,
		StoreName:    store,
		SourceImage:  imageName,
	}).Parse(boxes)
}
