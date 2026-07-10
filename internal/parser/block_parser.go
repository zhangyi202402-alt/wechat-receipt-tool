package parser

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kalading/wechat-receipt-tool/internal/models"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

const (
	defaultRowGap    = 18.0
	defaultBlockGap  = 45.0
	lowScoreThreshold = 0.75
)

var (
	monthHeaderRe = regexp.MustCompile(`(\d{4})年(\d{1,2})月`)
	sourceRe      = regexp.MustCompile(`二维码收款[-—]来自(\S+)`)
	amountRe      = regexp.MustCompile(`\+(\d+(?:\.\d{1,2})?)`)
	dateTimeRe    = regexp.MustCompile(`(\d{1,2})月(\d{1,2})日\s*(\d{1,2}):(\d{2})`)
)

type visualLine struct {
	text   string
	topY   float64
	bottom float64
	score  float64
}

type block struct {
	lines []visualLine
}

type Options struct {
	FallbackYear  int
	StoreName     string
	SourceImage   string
	RowGap        float64
	BlockGap      float64
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
		if monthHeaderRe.MatchString(combined) && !strings.Contains(combined, "二维码收款") {
			continue
		}
		if !strings.Contains(combined, "二维码收款") {
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

	if m := sourceRe.FindStringSubmatch(combined); len(m) == 2 {
		rec.Source = strings.TrimSpace(m[1])
	} else {
		return rec, false
	}

	fullAmt, hasFull := parseAmountFromText(combined)
	colAmt, hasCol := bestAmountFromColumn(blk, allBoxes)
	if amt, ok := mergeAmountChoice(fullAmt, colAmt, hasFull, hasCol); ok {
		rec.Amount = amt
	} else {
		return rec, false
	}

	if m := dateTimeRe.FindStringSubmatch(combined); len(m) == 5 {
		mo, _ := strconv.Atoi(m[1])
		day, _ := strconv.Atoi(m[2])
		hour, _ := strconv.Atoi(m[3])
		minute, _ := strconv.Atoi(m[4])
		useYear := year
		if useYear == 0 {
			useYear = p.opts.FallbackYear
		}
		if useYear == 0 {
			useYear = time.Now().Year()
		}
		if month == 0 {
			month = mo
		}
		rec.Date = fmt.Sprintf("%04d-%02d-%02d", useYear, mo, day)
		rec.Time = fmt.Sprintf("%02d:%02d", hour, minute)
	} else {
		return rec, false
	}

	var minScore float64 = 1
	for _, ln := range blk.lines {
		if ln.score < minScore {
			minScore = ln.score
		}
	}
	rec.OCRScore = minScore
	rec.LowConfidence = minScore < lowScoreThreshold
	return rec, true
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
		var parts []string
		seenText := make(map[string]struct{})
		var top, bottom, scoreSum float64
		var count float64
		for i, b := range g {
			if _, ok := seenText[b.Text]; ok {
				continue
			}
			seenText[b.Text] = struct{}{}
			parts = append(parts, b.Text)
			t, bt := boxTop(b.Box), boxBottom(b.Box)
			if i == 0 || t < top {
				top = t
			}
			if bt > bottom {
				bottom = bt
			}
			scoreSum += b.Score
			count++
		}
		if count == 0 {
			continue
		}
		lines = append(lines, visualLine{
			text:   strings.Join(parts, " "),
			topY:   top,
			bottom: bottom,
			score:  scoreSum / count,
		})
	}
	return lines
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
		if strings.Contains(cur.text, "二维码收款") {
			src := qrSourceKey(cur.text)
			if len(current.lines) > 0 {
				prevCombined := strings.Join(lineTexts(current.lines), " ")
				prevSrc := qrSourceKey(prevCombined)
				if src != "" && src == prevSrc {
					gap := cur.topY - lineAnchorY(current.lines)
					if gap >= 0 && gap <= receiptLineGap {
						current.lines = appendUniqueLine(current.lines, cur)
						continue
					}
				}
				flush()
			}
			current.lines = []visualLine{cur}
			continue
		}
		if len(current.lines) == 0 {
			if strings.Contains(cur.text, "退款") || strings.Contains(cur.text, "提现") || strings.Contains(cur.text, "转账") {
				current.lines = []visualLine{cur}
			}
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
		return out[i].Time < out[j].Time
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
	Text  string       `json:"text"`
	Box   [4][2]float64 `json:"box"`
	Score float64      `json:"score"`
}
