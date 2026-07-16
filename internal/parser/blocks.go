package parser

import (
	"strings"
)

const receiptLineGap = 40.0

func appendUniqueLine(lines []visualLine, ln visualLine) []visualLine {
	for _, existing := range lines {
		if existing.text == ln.text {
			return lines
		}
	}
	return append(lines, ln)
}

func mergeBlockLines(a, b block) block {
	out := block{lines: append([]visualLine(nil), a.lines...)}
	for _, ln := range b.lines {
		out.lines = appendUniqueLine(out.lines, ln)
	}
	return out
}

func mergeReceiptBlocks(blocks []block) []block {
	if len(blocks) == 0 {
		return nil
	}
	var out []block
	for _, b := range blocks {
		combined := strings.Join(lineTexts(b.lines), " ")
		key := txMergeKey(combined)
		if key != "" && len(out) > 0 {
			prev := out[len(out)-1]
			prevCombined := strings.Join(lineTexts(prev.lines), " ")
			if txMergeKey(prevCombined) == key {
				gap := blockAnchorY(b) - blockAnchorY(prev)
				if gap >= 0 && gap <= receiptLineGap {
					out[len(out)-1] = mergeBlockLines(prev, b)
					continue
				}
			}
		}
		out = append(out, b)
	}
	return out
}

func blockAnchorY(b block) float64 {
	return lineAnchorY(b.lines)
}

func lineAnchorY(lines []visualLine) float64 {
	for _, ln := range lines {
		if isTxTitle(ln.text) || hasSignedAmount(ln.text) {
			return ln.topY
		}
	}
	if len(lines) == 0 {
		return 0
	}
	return lines[len(lines)-1].topY
}

func isTransactionAnchor(text string) bool {
	if isMonthSummaryLine(text) {
		return false
	}
	if monthHeaderRe.MatchString(text) {
		return false
	}
	return hasSignedAmount(text) || isTxTitle(text)
}
