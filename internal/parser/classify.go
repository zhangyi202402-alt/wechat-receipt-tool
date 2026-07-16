package parser

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/kalading/wechat-receipt-tool/internal/models"
)

var (
	qrSourceRe     = regexp.MustCompile(`二维码收款[-—]来自(\S+)`)
	transferFromRe = regexp.MustCompile(`转账[-—]来自(\S+)`)
	transferToRe   = regexp.MustCompile(`转账[-—]转给(\S+)`)
	redPacketViaRe = regexp.MustCompile(`(?:微信)?红包[-—](.+)`)
)

type classified struct {
	Type         models.TxType
	Title        string
	Source       string
	Classified   bool
	SourceTrunc  bool // 对方可能被截断
}

func classifyBlock(combined string, lines []visualLine) classified {
	title := bestTitleLine(lines, combined)
	c := classified{Title: title, Classified: true}

	fullTexts := fullImageTexts(lines)
	searchPool := append([]string{title, combined}, fullTexts...)

	switch {
	case strings.Contains(title, "二维码收款") || containsAny(fullTexts, "二维码收款") || strings.Contains(combined, "二维码收款"):
		c.Type = models.TxQRReceipt
		c.Source = longestCapture(searchPool, qrSourceRe)
		if c.Source == "" {
			c.Source = title
		}
	case strings.Contains(title, "转账") || containsAny(fullTexts, "转账") || strings.Contains(combined, "转账"):
		if src := longestCapture(searchPool, transferToRe); src != "" ||
			strings.Contains(title, "转给") || containsAny(fullTexts, "转给") {
			c.Type = models.TxTransferOut
			c.Source = src
			if c.Source == "" {
				c.Source = title
			}
		} else {
			c.Type = models.TxTransferIn
			c.Source = longestCapture(searchPool, transferFromRe)
			if c.Source == "" {
				c.Source = title
			}
		}
	case strings.Contains(title, "微信红包") || strings.Contains(title, "红包") ||
		containsAny(fullTexts, "红包") || strings.Contains(combined, "红包"):
		c.Type = models.TxRedPacket
		if m := redPacketViaRe.FindStringSubmatch(title); len(m) == 2 {
			c.Source = strings.TrimSpace(m[1])
		} else {
			c.Source = title
		}
	case strings.Contains(title, "退款") || containsAny(fullTexts, "退款") || strings.Contains(combined, "退款"):
		c.Type = models.TxRefund
		c.Source = cleanMerchantTitle(title)
	case strings.Contains(title, "提现") || containsAny(fullTexts, "提现") || strings.Contains(combined, "提现"):
		c.Type = models.TxWithdraw
		c.Source = cleanMerchantTitle(title)
	case isTxTitle(title):
		c.Type = models.TxOther
		c.Source = title
		c.Classified = false
	default:
		c.Type = models.TxMerchant
		c.Source = cleanMerchantTitle(title)
		if c.Source == "" {
			c.Classified = false
			c.Type = models.TxOther
		}
	}

	c.Source = truncateSource(c.Source)
	c.Title = cleanMerchantTitle(c.Title)
	if c.Title == "" {
		c.Title = c.Source
	}
	c.SourceTrunc = isLikelyTruncatedSource(c.Type, c.Source, c.Title)
	return c
}

func containsAny(texts []string, sub string) bool {
	for _, t := range texts {
		if strings.Contains(t, sub) {
			return true
		}
	}
	return false
}

func fullImageTexts(lines []visualLine) []string {
	var out []string
	for _, ln := range lines {
		if ln.fromAmount || ln.fromTime && !ln.fromFull {
			continue
		}
		t := strings.TrimSpace(ln.text)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// bestTitleLine scores candidate title lines; prefers full-image, longer, with type prefix, without bare datetime.
func bestTitleLine(lines []visualLine, combined string) string {
	type cand struct {
		text  string
		score int
	}
	var best cand
	consider := func(t string, fromFull bool, ocrScore float64) {
		t = strings.TrimSpace(t)
		if t == "" || monthHeaderRe.MatchString(t) || isMonthSummaryLine(t) {
			return
		}
		if amountRe.MatchString(t) && stripAmounts(t) == "" {
			return
		}
		if dateTimeRe.MatchString(t) && !isTxTitle(t) && stripDateTimes(t) == "" {
			return
		}
		cleaned := cleanTitle(stripAmounts(t))
		cleaned = cleanMerchantTitle(cleaned)
		if cleaned == "" {
			return
		}
		s := utf8.RuneCountInString(cleaned) * 10
		if fromFull {
			s += 50
		}
		if isTxTitle(cleaned) {
			s += 40
		}
		if strings.Contains(cleaned, "来自") || strings.Contains(cleaned, "转给") {
			s += 20
		}
		if dateTimeRe.MatchString(t) {
			s -= 15 // 标题+日期合并行略降权，纯标题优先
		}
		s += int(ocrScore * 10)
		if s > best.score {
			best = cand{text: cleaned, score: s}
		}
	}

	for _, ln := range lines {
		// 列二次 OCR 碎片不单独作标题，除非没有整图行
		if (ln.fromTime || ln.fromAmount) && !ln.fromFull {
			continue
		}
		consider(ln.text, ln.fromFull || (!ln.fromTime && !ln.fromAmount), ln.score)
	}
	if best.text != "" {
		return best.text
	}
	// fallback: 允许时间列参与
	for _, ln := range lines {
		consider(ln.text, false, ln.score)
	}
	if best.text != "" {
		return best.text
	}
	return cleanMerchantTitle(stripAmounts(stripDateTimes(combined)))
}

func longestCapture(texts []string, re *regexp.Regexp) string {
	best := ""
	for _, t := range texts {
		for _, m := range re.FindAllStringSubmatch(t, -1) {
			if len(m) < 2 {
				continue
			}
			src := strings.TrimSpace(m[1])
			// 去掉误拼进的日期，保留省略号
			src = strings.TrimSpace(dateTimeRe.ReplaceAllString(src, ""))
			if utf8.RuneCountInString(src) > utf8.RuneCountInString(best) {
				best = src
			}
		}
	}
	return best
}

func cleanMerchantTitle(s string) string {
	s = cleanTitle(s)
	s = stripDateTimes(s)
	s = stripAmounts(s)
	// 去掉重复片段（OCR 整图+时间列拼接）
	parts := strings.Fields(s)
	var out []string
	seen := make(map[string]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		// 子串被更长片段覆盖则跳过
		covered := false
		for _, e := range out {
			if strings.Contains(e, p) && e != p {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		// 新片段覆盖已有子串则替换
		for i := len(out) - 1; i >= 0; i-- {
			if strings.Contains(p, out[i]) && p != out[i] {
				out = append(out[:i], out[i+1:]...)
			}
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, "")
}

func cleanTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func stripAmounts(s string) string {
	return strings.TrimSpace(amountRe.ReplaceAllString(s, ""))
}

func stripDateTimes(s string) string {
	return strings.TrimSpace(dateTimeRe.ReplaceAllString(s, ""))
}

func truncateSource(s string) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > 40 {
		return string(runes[:40]) + "…"
	}
	return s
}

func isLikelyTruncatedSource(txType models.TxType, source, title string) bool {
	if source == "" {
		return true
	}
	n := utf8.RuneCountInString(source)
	switch txType {
	case models.TxQRReceipt, models.TxTransferIn, models.TxTransferOut:
		if n <= 1 {
			return true
		}
		// 标题里来自后的片段明显更长
		if m := qrSourceRe.FindStringSubmatch(title); len(m) == 2 {
			full := strings.TrimSpace(m[1])
			if utf8.RuneCountInString(full) > n+1 {
				return true
			}
		}
	}
	return false
}

func directionFromAmount(amount float64, txType models.TxType) string {
	switch txType {
	case models.TxTransferOut, models.TxWithdraw:
		if amount > 0 {
			return models.DirectionOut
		}
	}
	if amount < 0 {
		return models.DirectionOut
	}
	return models.DirectionIn
}

func isTxTitle(text string) bool {
	return strings.Contains(text, "二维码收款") ||
		strings.Contains(text, "转账") ||
		strings.Contains(text, "微信红包") ||
		strings.Contains(text, "红包") ||
		strings.Contains(text, "退款") ||
		strings.Contains(text, "提现")
}

func hasSignedAmount(text string) bool {
	return amountRe.MatchString(text)
}

func isMonthSummaryLine(text string) bool {
	if dateTimeRe.MatchString(text) || strings.Contains(text, "日") {
		return false
	}
	if strings.Contains(text, "支出") || strings.Contains(text, "收入") {
		return true
	}
	return false
}

func txMergeKey(text string) string {
	if m := qrSourceRe.FindStringSubmatch(text); len(m) == 2 {
		return "qr:" + strings.TrimSpace(m[1])
	}
	if m := transferToRe.FindStringSubmatch(text); len(m) == 2 {
		return "to:" + strings.TrimSpace(m[1])
	}
	if m := transferFromRe.FindStringSubmatch(text); len(m) == 2 {
		return "from:" + strings.TrimSpace(m[1])
	}
	if isTxTitle(text) {
		return "title:" + cleanTitle(stripAmounts(stripDateTimes(text)))
	}
	return ""
}

func lineRole(text string) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return "noise"
	}
	if monthHeaderRe.MatchString(t) || isMonthSummaryLine(t) {
		return "noise"
	}
	if amountRe.MatchString(t) && stripAmounts(t) == "" {
		return "amount"
	}
	if dateTimeRe.MatchString(t) && !isTxTitle(t) && stripDateTimes(t) == "" {
		return "datetime"
	}
	if isTxTitle(t) || (!hasSignedAmount(t) && !dateTimeRe.MatchString(t)) {
		return "title"
	}
	if isTxTitle(t) {
		return "title"
	}
	return "title"
}
