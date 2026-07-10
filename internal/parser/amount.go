package parser

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

var amountFullRe = regexp.MustCompile(`\+(\d+\.\d{2})`)

func bestAmountFromColumn(blk block, allBoxes []ocr.TextBox) (float64, bool) {
	top, bottom := blockYRange(blk)
	const yPad = 12.0

	var bestVal float64
	var bestRank int
	found := false

	for _, b := range allBoxes {
		if !b.AmountColumn {
			continue
		}
		cy := boxCenterY(b.Box)
		if cy < top-yPad || cy > bottom+yPad {
			continue
		}
		for _, cand := range amountCandidates(b.Text) {
			rank := amountRank(cand.raw, cand.value)
			if !found || rank > bestRank {
				bestVal = cand.value
				bestRank = rank
				found = true
			}
		}
	}
	return bestVal, found
}

type amountCand struct {
	raw   string
	value float64
}

func amountCandidates(text string) []amountCand {
	var out []amountCand
	for _, m := range amountRe.FindAllStringSubmatch(text, -1) {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		out = append(out, amountCand{raw: m[1], value: v})
	}
	return out
}

// amountRank: 优先完整两位小数，其次整数部分位数更多（+300 优于 +30）
func amountRank(raw string, value float64) int {
	rank := int(value * 100)
	if amountFullRe.MatchString("+" + raw) {
		rank += 100000
	}
	if i := strings.Index(raw, "."); i >= 0 {
		rank += len(raw[:i]) * 1000
	} else {
		rank += len(raw) * 1000
	}
	return rank
}

func parseAmountFromText(text string) (float64, bool) {
	m := amountRe.FindStringSubmatch(text)
	if len(m) != 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func blockYRange(blk block) (top, bottom float64) {
	if len(blk.lines) == 0 {
		return 0, 0
	}
	top = blk.lines[0].topY
	bottom = blk.lines[0].bottom
	for _, ln := range blk.lines[1:] {
		if ln.topY < top {
			top = ln.topY
		}
		if ln.bottom > bottom {
			bottom = ln.bottom
		}
	}
	return top, bottom
}

func boxCenterY(box [4][2]float64) float64 {
	return (box[0][1] + box[2][1]) / 2
}

func boxCenterX(box [4][2]float64) float64 {
	return (box[0][0] + box[2][0]) / 2
}

func mergeAmountChoice(fullAmount, colAmount float64, hasFull, hasCol bool) (float64, bool) {
	if hasCol && hasFull && !floatEqual(fullAmount, colAmount) {
		return colAmount, true
	}
	if hasCol {
		return colAmount, true
	}
	return fullAmount, hasFull
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}
