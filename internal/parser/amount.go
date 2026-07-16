package parser

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

var amountFullRe = regexp.MustCompile(`[+-]\d+\.\d{2}`)

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
		v, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			continue
		}
		if m[1] == "-" {
			v = -v
		}
		out = append(out, amountCand{raw: m[1] + m[2], value: v})
	}
	return out
}

// amountRank: 优先完整两位小数，其次绝对值整数部分位数更多（+300 优于 +30）
func amountRank(raw string, value float64) int {
	abs := math.Abs(value)
	rank := int(abs * 100)
	if amountFullRe.MatchString(raw) {
		rank += 100000
	}
	numPart := strings.TrimLeft(raw, "+-")
	if i := strings.Index(numPart, "."); i >= 0 {
		rank += len(numPart[:i]) * 1000
	} else {
		rank += len(numPart) * 1000
	}
	return rank
}

func parseAmountFromText(text string) (float64, bool) {
	cands := amountCandidates(text)
	if len(cands) == 0 {
		return 0, false
	}
	best := cands[0]
	bestRank := amountRank(best.raw, best.value)
	for _, c := range cands[1:] {
		if r := amountRank(c.raw, c.value); r > bestRank {
			best = c
			bestRank = r
		}
	}
	return best.value, true
}

func formatSignedAmount(v float64) string {
	if v < 0 {
		return fmt.Sprintf("%.2f", v)
	}
	return fmt.Sprintf("+%.2f", v)
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

func mergeAmountChoice(fullAmount, colAmount float64, hasFull, hasCol bool) (float64, bool, bool) {
	if !hasCol && !hasFull {
		return 0, false, false
	}
	if hasCol && hasFull && !floatEqual(fullAmount, colAmount) {
		// 整图绝对值更大且带更完整数字时优先整图（防 +100 被金额列 +10 覆盖）
		if math.Abs(fullAmount) > math.Abs(colAmount) {
			fullRank := amountRank(formatSignedAmount(fullAmount), fullAmount)
			colRank := amountRank(formatSignedAmount(colAmount), colAmount)
			if fullRank >= colRank {
				return fullAmount, true, true
			}
		}
		return colAmount, true, true
	}
	if hasCol {
		return colAmount, true, false
	}
	return fullAmount, hasFull, false
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}
