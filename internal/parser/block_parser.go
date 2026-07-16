package parser

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kalading/wechat-receipt-tool/internal/models"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

const (
	defaultRowGap   = 18.0
	defaultBlockGap = 45.0
	defaultReviewConf = 0.85
)

var (
	monthHeaderRe = regexp.MustCompile(`(\d{4})年(\d{1,2})月`)
	amountRe      = regexp.MustCompile(`([+-])(\d+(?:\.\d{1,2})?)`)
	dateTimeRe    = regexp.MustCompile(`(\d{1,2})月(\d{1,2})日\s*(\d{1,2}):(\d{2})`)
)

type visualLine struct {
	text       string
	topY       float64
	bottom     float64
	score      float64
	left       float64
	right      float64
	fromFull   bool // 含整图 OCR 框
	fromTime   bool // 含时间列二次 OCR
	fromAmount bool // 含金额列二次 OCR
}


type block struct {
	lines []visualLine
}

type Options struct {
	FallbackYear            int
	StoreName               string
	SourceImage             string
	ImageWidth              float64
	RowGap                  float64
	BlockGap                float64
	RequireDate             bool
	ReviewConfidenceBelow   float64
}

type BlockParser struct {
	opts Options
}

func NewBlockParser(opts Options) *BlockParser {
	if opts.RowGap <= 0 {
		opts.RowGap = defaultRowGap
	}
	if opts.BlockGap <= 0 {
		opts.BlockGap = defaultBlockGap
	}
	if opts.ReviewConfidenceBelow <= 0 {
		opts.ReviewConfidenceBelow = defaultReviewConf
	}
	return &BlockParser{opts: opts}
}

func (p *BlockParser) Parse(boxes []ocr.TextBox) []models.ReceiptRecord {
	if len(boxes) == 0 {
		return nil
	}
	boxes = ocr.DeduplicateBoxes(boxes)
	lines := clusterLines(boxes, p.opts.RowGap)
	blocks := mergeReceiptBlocks(clusterBlocks(lines, p.opts.BlockGap))

	year, month := p.opts.FallbackYear, 0
	var records []models.ReceiptRecord

	for _, blk := range blocks {
		for _, ln := range blk.lines {
			if m := monthHeaderRe.FindStringSubmatch(ln.text); len(m) == 3 {
				year, _ = strconv.Atoi(m[1])
				month, _ = strconv.Atoi(m[2])
			}
		}
		combined := strings.Join(lineTexts(blk.lines), " ")
		if monthHeaderRe.MatchString(combined) && !hasSignedAmount(combined) && !isTxTitle(combined) {
			continue
		}
		if isMonthSummaryLine(combined) && !hasSignedAmount(combined) {
			continue
		}
		if !hasSignedAmount(combined) && !isTxTitle(combined) {
			continue
		}
		rec, ok := p.extractReceipt(blk, boxes, year, month)
		if !ok {
			continue
		}
		rec.Transferor = p.opts.StoreName
		rec.SourceImage = p.opts.SourceImage
		rec.ParsedAt = time.Now()
		records = append(records, rec)
	}
	return records
}

func (p *BlockParser) extractReceipt(blk block, allBoxes []ocr.TextBox, year, month int) (models.ReceiptRecord, bool) {
	combined := strings.Join(lineTexts(blk.lines), "\n")
	var rec models.ReceiptRecord
	var reasons []string
	var missing []string

	cls := classifyBlock(combined, blk.lines)
	rec.Type = cls.Type
	rec.Title = cls.Title
	rec.Source = cls.Source
	if rec.Source == "" {
		missing = append(missing, "source")
		reasons = append(reasons, "缺对方")
	}
	if cls.SourceTrunc {
		reasons = append(reasons, "对方可能截断")
	}
	if !cls.Classified || cls.Type == models.TxOther {
		reasons = append(reasons, "类型无法归类")
	}

	fullAmt, hasFull := parseAmountFromText(combined)
	colAmt, hasCol := bestAmountFromColumn(blk, allBoxes)
	amt, hasAmt, conflict := mergeAmountChoice(fullAmt, colAmt, hasFull, hasCol)
	if !hasAmt {
		missing = append(missing, "amount")
		reasons = append(reasons, "缺金额")
		return rec, false
	}
	rec.Amount = amt
	if conflict {
		reasons = append(reasons, fmt.Sprintf("金额列与整图冲突(%s vs %s)",
			formatSignedAmount(fullAmt), formatSignedAmount(colAmt)))
	}

	rec.Direction = directionFromAmount(rec.Amount, rec.Type)

	dt, hasDT := bestDateTime(combined, blk, allBoxes)
	singleDigitHour := false
	if hasDT {
		useYear := year
		if useYear == 0 {
			useYear = p.opts.FallbackYear
		}
		dateStr, timeStr, sdh := formatDateTime(dt, useYear, p.opts.FallbackYear)
		rec.Date = dateStr
		rec.Time = timeStr
		singleDigitHour = sdh
		if singleDigitHour {
			reasons = append(reasons, "时间低置信(一位数小时)")
		}
	} else {
		missing = append(missing, "date")
		reasons = append(reasons, "缺日期")
		if p.opts.RequireDate {
			return rec, false
		}
	}

	var minScore float64 = 1
	for _, ln := range blk.lines {
		if ln.score < minScore {
			minScore = ln.score
		}
	}
	rec.OCRScore = minScore
	rec.BandBox = computeBandBox(blk, p.opts.ImageWidth)

	conf := minScore
	for range missing {
		conf -= 0.12
	}
	if conflict {
		conf -= 0.1
	}
	if singleDigitHour {
		conf -= 0.08
	}
	if !cls.Classified {
		conf -= 0.05
	}
	if cls.SourceTrunc {
		conf -= 0.05
	}
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}
	rec.Confidence = conf

	threshold := p.opts.ReviewConfidenceBelow
	if conf < threshold {
		reasons = append(reasons, "置信度低于阈值")
	}

	rec.Missing = missing
	rec.ReviewReasons = uniqueStrings(reasons)
	if len(rec.ReviewReasons) > 0 {
		rec.Status = models.StatusNeedsReview
		rec.LowConfidence = true
	} else {
		rec.Status = models.StatusOK
	}

	// 有金额即可出；标题可空但尽量有
	if rec.Title == "" && rec.Source == "" {
		return rec, false
	}
	return rec, true
}

func computeBandBox(blk block, imgW float64) [4][2]float64 {
	top, bottom := blockYRange(blk)
	const pad = 8.0
	top -= pad
	bottom += pad
	if top < 0 {
		top = 0
	}
	left, right := 0.0, imgW
	if imgW <= 0 {
		left = math.MaxFloat64
		right = 0
		for _, ln := range blk.lines {
			if ln.left < left {
				left = ln.left
			}
			if ln.right > right {
				right = ln.right
			}
		}
		if left == math.MaxFloat64 {
			left = 0
		}
		if right < left+10 {
			right = left + 400
		}
		// 左右再扩一点
		left -= 20
		if left < 0 {
			left = 0
		}
		right += 20
	}
	return [4][2]float64{
		{left, top},
		{right, top},
		{right, bottom},
		{left, bottom},
	}
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func clusterLines(boxes []ocr.TextBox, rowGap float64) []visualLine {
	if len(boxes) == 0 {
		return nil
	}
	sorted := append([]ocr.TextBox(nil), boxes...)
	sort.Slice(sorted, func(i, j int) bool {
		cyi := lineCenterY(sorted[i])
		cyj := lineCenterY(sorted[j])
		if math.Abs(cyi-cyj) < 1 {
			return lineCenterX(sorted[i]) < lineCenterX(sorted[j])
		}
		return cyi < cyj
	})

	var groups [][]ocr.TextBox
	current := []ocr.TextBox{sorted[0]}
	refY := lineCenterY(sorted[0])
	for i := 1; i < len(sorted); i++ {
		cy := lineCenterY(sorted[i])
		if math.Abs(cy-refY) <= rowGap {
			current = append(current, sorted[i])
		} else {
			groups = append(groups, current)
			current = []ocr.TextBox{sorted[i]}
			refY = cy
		}
	}
	groups = append(groups, current)

	var lines []visualLine
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool {
			return lineCenterX(g[i]) < lineCenterX(g[j])
		})
		ln := mergeBoxesToLine(g)
		if ln.text == "" {
			continue
		}
		lines = append(lines, ln)
	}
	return lines
}

type linePiece struct {
	text  string
	score float64
	full  bool
	time  bool
	amt   bool
}

// mergeBoxesToLine builds one visual line from same-Y boxes.
// Prefers full-image text; drops shorter column fragments that are prefixes/near-duplicates.
func mergeBoxesToLine(g []ocr.TextBox) visualLine {
	var top, bottom, scoreSum, left, right float64
	var count float64
	left = math.MaxFloat64
	fromFull, fromTime, fromAmount := false, false, false

	var pieces []linePiece

	for i, b := range g {
		t, bt := boxTop(b.Box), boxBottom(b.Box)
		l, r := boxLeft(b.Box), boxRight(b.Box)
		if i == 0 || t < top {
			top = t
		}
		if bt > bottom {
			bottom = bt
		}
		if l < left {
			left = l
		}
		if r > right {
			right = r
		}
		scoreSum += b.Score
		count++
		full := !b.AmountColumn && !b.TimeColumn
		if full {
			fromFull = true
		}
		if b.TimeColumn {
			fromTime = true
		}
		if b.AmountColumn {
			fromAmount = true
		}
		pieces = append(pieces, linePiece{
			text: b.Text, score: b.Score, full: full, time: b.TimeColumn, amt: b.AmountColumn,
		})
	}
	if count == 0 {
		return visualLine{}
	}
	if left == math.MaxFloat64 {
		left = 0
	}

	selected := selectLineTexts(pieces)
	return visualLine{
		text:       strings.Join(selected, " "),
		topY:       top,
		bottom:     bottom,
		score:      scoreSum / count,
		left:       left,
		right:      right,
		fromFull:   fromFull,
		fromTime:   fromTime,
		fromAmount: fromAmount,
	}
}

func selectLineTexts(pieces []linePiece) []string {
	var fulls, amts, times []linePiece
	for _, p := range pieces {
		switch {
		case p.amt:
			amts = append(amts, p)
		case p.time:
			times = append(times, p)
		default:
			fulls = append(fulls, p)
		}
	}

	var out []string
	addUniqueLongest := func(list []linePiece) {
		for _, p := range list {
			t := strings.TrimSpace(p.text)
			if t == "" {
				continue
			}
			dup := false
			for i, e := range out {
				if e == t {
					dup = true
					break
				}
				if strings.Contains(e, t) {
					dup = true
					break
				}
				if strings.Contains(t, e) && utf8.RuneCountInString(t) > utf8.RuneCountInString(e) {
					out[i] = t
					dup = true
					break
				}
				if textNearCover(e, t) {
					if utf8.RuneCountInString(t) > utf8.RuneCountInString(e) {
						out[i] = t
					}
					dup = true
					break
				}
			}
			if !dup {
				out = append(out, t)
			}
		}
	}

	addUniqueLongest(fulls)
	addUniqueLongest(amts)
	for _, p := range times {
		t := strings.TrimSpace(p.text)
		if t == "" {
			continue
		}
		if dateTimeRe.MatchString(t) && stripDateTimes(t) == "" {
			already := false
			for _, e := range out {
				if strings.Contains(e, t) || dateTimeRe.MatchString(e) {
					already = true
					break
				}
			}
			if !already {
				out = append(out, t)
			}
			continue
		}
		covered := false
		for _, e := range out {
			if strings.Contains(e, t) || textNearCover(e, t) || (strings.Contains(t, e) && isTxTitle(e)) {
				if utf8.RuneCountInString(e) >= utf8.RuneCountInString(t) {
					covered = true
					break
				}
			}
		}
		if !covered && isTxTitle(t) {
			hasTitle := false
			for _, e := range out {
				if isTxTitle(e) {
					hasTitle = true
					break
				}
			}
			if !hasTitle {
				out = append(out, t)
			}
		}
	}
	return out
}

func textNearCover(longer, shorter string) bool {
	a := strings.TrimRight(longer, "….")
	b := strings.TrimRight(shorter, "….")
	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}
	if isTxTitle(a) && isTxTitle(b) {
		if strings.Contains(a, "二维码收款") && strings.Contains(b, "二维码收款") {
			return true
		}
		if strings.Contains(a, "转账") && strings.Contains(b, "转账") {
			return true
		}
	}
	return false
}

func clusterBlocks(lines []visualLine, blockGap float64) []block {
	if len(lines) == 0 {
		return nil
	}
	var blocks []block
	var current block

	flush := func() {
		if len(current.lines) > 0 {
			blocks = append(blocks, current)
			current = block{}
		}
	}

	for _, cur := range lines {
		if monthHeaderRe.MatchString(cur.text) {
			flush()
			blocks = append(blocks, block{lines: []visualLine{cur}})
			continue
		}
		if isMonthSummaryLine(cur.text) {
			flush()
			continue
		}
		if isTransactionAnchor(cur.text) {
			// 纯金额行：并入当前交易带（标题与金额常分行）
			if hasSignedAmount(cur.text) && !isTxTitle(cur.text) && len(current.lines) > 0 {
				gap := cur.topY - lineAnchorY(current.lines)
				if gap >= -5 && gap <= receiptLineGap {
					current.lines = appendUniqueLine(current.lines, cur)
					continue
				}
			}
			key := txMergeKey(cur.text)
			if len(current.lines) > 0 {
				prevCombined := strings.Join(lineTexts(current.lines), " ")
				prevKey := txMergeKey(prevCombined)
				if key != "" && key == prevKey {
					gap := cur.topY - lineAnchorY(current.lines)
					if gap >= 0 && gap <= receiptLineGap {
						current.lines = appendUniqueLine(current.lines, cur)
						continue
					}
				}
				// 无 key 的标题行（商户名）后跟金额已在上面处理；
				// 时间行不是 anchor。此处新开块。
				flush()
			}
			current.lines = []visualLine{cur}
			continue
		}
		if len(current.lines) == 0 {
			continue
		}
		prev := current.lines[len(current.lines)-1]
		gap := cur.topY - prev.bottom
		if gap > blockGap {
			flush()
			continue
		}
		current.lines = append(current.lines, cur)
	}
	flush()
	return blocks
}

func lineTexts(lines []visualLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.text
	}
	return out
}

func lineCenterY(b ocr.TextBox) float64 {
	return (b.Box[0][1] + b.Box[1][1] + b.Box[2][1] + b.Box[3][1]) / 4
}

func lineCenterX(b ocr.TextBox) float64 {
	return (b.Box[0][0] + b.Box[1][0] + b.Box[2][0] + b.Box[3][0]) / 4
}

func boxTop(box [4][2]float64) float64 {
	minY := box[0][1]
	for i := 1; i < 4; i++ {
		if box[i][1] < minY {
			minY = box[i][1]
		}
	}
	return minY
}

func boxBottom(box [4][2]float64) float64 {
	maxY := box[0][1]
	for i := 1; i < 4; i++ {
		if box[i][1] > maxY {
			maxY = box[i][1]
		}
	}
	return maxY
}

func boxLeft(box [4][2]float64) float64 {
	minX := box[0][0]
	for i := 1; i < 4; i++ {
		if box[i][0] < minX {
			minX = box[i][0]
		}
	}
	return minX
}

func boxRight(box [4][2]float64) float64 {
	maxX := box[0][0]
	for i := 1; i < 4; i++ {
		if box[i][0] > maxX {
			maxX = box[i][0]
		}
	}
	return maxX
}

func Deduplicate(records []models.ReceiptRecord) []models.ReceiptRecord {
	seen := make(map[models.DedupKey]models.ReceiptRecord)
	for _, r := range records {
		key := r.Key()
		if existing, ok := seen[key]; ok {
			if r.OCRScore > existing.OCRScore {
				seen[key] = r
			}
			continue
		}
		seen[key] = r
	}
	out := make([]models.ReceiptRecord, 0, len(seen))
	for _, r := range seen {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		if out[i].Time != out[j].Time {
			return out[i].Time < out[j].Time
		}
		return out[i].Amount < out[j].Amount
	})
	for i := range out {
		out[i].Serial = i + 1
	}
	return out
}

// ParseFromLines is a test helper that builds synthetic boxes from fixture lines.
func ParseFromLines(lines []FixtureLine, opts Options) []models.ReceiptRecord {
	var boxes []ocr.TextBox
	for _, ln := range lines {
		boxes = append(boxes, ocr.TextBox{
			Text:  ln.Text,
			Box:   ln.Box,
			Score: ln.Score,
		})
	}
	return NewBlockParser(opts).Parse(boxes)
}

type FixtureLine struct {
	Text  string        `json:"text"`
	Box   [4][2]float64 `json:"box"`
	Score float64       `json:"score"`
}
