package parser

import (
	"strings"
)

const receiptBandGap = 85.0
const receiptLineGap = 40.0

func qrSourceKey(text string) string {
	if m := sourceRe.FindStringSubmatch(text); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

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
		if !strings.Contains(combined, "二维码收款") {
			out = append(out, b)
			continue
		}
		src := qrSourceKey(combined)
		if src != "" && len(out) > 0 {
			prev := out[len(out)-1]
			prevCombined := strings.Join(lineTexts(prev.lines), " ")
			if strings.Contains(prevCombined, "二维码收款") && qrSourceKey(prevCombined) == src {
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
		if strings.Contains(ln.text, "二维码收款") {
			return ln.topY
		}
	}
	return lines[len(lines)-1].topY
}
