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
		}
		fmt.Fprintf(w, "  %3d y=%6.0f x=%6.0f score=%.3f%s %q\n",
			i+1, line.y, line.x, line.score, tag, line.text)
	}
	fmt.Fprintf(w, "解析记录 (%d):\n", len(records))
	if len(records) == 0 {
		fmt.Fprintln(w, "  (无)")
	} else {
		for _, r := range records {
			conf := ""
			if r.LowConfidence {
				conf = " [低置信度]"
			}
			fmt.Fprintf(w, "  来源=%s 金额=%.2f 日期=%s %s OCRScore=%.3f%s\n",
				r.Source, r.Amount, r.Date, r.Time, r.OCRScore, conf)
		}
	}
	fmt.Fprintln(w, "交易块预览:")
	for i, block := range previewQRBlocks(boxes) {
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
		}
	}
	return out
}

type boxLine struct {
	text      string
	x, y      float64
	score     float64
	amountCol bool
}

func previewQRBlocks(boxes []ocr.TextBox) []string {
	sorted := append([]ocr.TextBox(nil), boxes...)
	sort.Slice(sorted, func(i, j int) bool {
		return centerY(sorted[i].Box) < centerY(sorted[j].Box)
	})
	var blocks []string
	for i, b := range sorted {
		if !strings.Contains(b.Text, "二维码收款") {
			continue
		}
		startY := centerY(b.Box)
		var parts []string
		for j := i; j < len(sorted); j++ {
			cy := centerY(sorted[j].Box)
			if j > i && cy-startY > 120 {
				break
			}
			if j > i && strings.Contains(sorted[j].Text, "二维码收款") {
				break
			}
			parts = append(parts, fmt.Sprintf("%q", sorted[j].Text))
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
