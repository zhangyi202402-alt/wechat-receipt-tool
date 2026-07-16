package parser

import (
	"fmt"
	"strconv"
	"time"

	"github.com/kalading/wechat-receipt-tool/internal/ocr"
)

type dateTimeParts struct {
	month, day, hour, minute int
	hourDigits               int
	fromTimeCol              bool
	score                    float64
}

func parseDateTimeParts(text string) (dateTimeParts, bool) {
	m := dateTimeRe.FindStringSubmatch(text)
	if len(m) != 5 {
		return dateTimeParts{}, false
	}
	mo, _ := strconv.Atoi(m[1])
	day, _ := strconv.Atoi(m[2])
	hour, _ := strconv.Atoi(m[3])
	minute, _ := strconv.Atoi(m[4])
	return dateTimeParts{
		month:      mo,
		day:        day,
		hour:       hour,
		minute:     minute,
		hourDigits: len(m[3]),
	}, true
}

func dateTimeRank(p dateTimeParts) int {
	rank := p.hourDigits*100000 + p.hour*100 + p.minute
	if p.fromTimeCol {
		rank += 50000
	}
	if p.score > 0 {
		rank += int(p.score * 100)
	}
	return rank
}

func bestDateTime(combined string, blk block, allBoxes []ocr.TextBox) (dateTimeParts, bool) {
	var best dateTimeParts
	found := false

	consider := func(p dateTimeParts) {
		if !found || dateTimeRank(p) > dateTimeRank(best) {
			best = p
			found = true
		}
	}

	if p, ok := parseDateTimeParts(combined); ok {
		consider(p)
	}
	for _, ln := range blk.lines {
		if p, ok := parseDateTimeParts(ln.text); ok {
			p.score = ln.score
			consider(p)
		}
	}

	top, bottom := blockYRange(blk)
	const yPad = 20.0
	for _, b := range allBoxes {
		if !b.TimeColumn {
			continue
		}
		cy := boxCenterY(b.Box)
		if cy < top-yPad || cy > bottom+yPad {
			continue
		}
		if p, ok := parseDateTimeParts(b.Text); ok {
			p.fromTimeCol = true
			p.score = b.Score
			consider(p)
		}
	}
	return best, found
}

func formatDateTime(p dateTimeParts, year, fallbackYear int) (date, tm string, singleDigitHour bool) {
	useYear := year
	if useYear == 0 {
		useYear = fallbackYear
	}
	if useYear == 0 {
		useYear = time.Now().Year()
	}
	date = fmt.Sprintf("%04d-%02d-%02d", useYear, p.month, p.day)
	tm = fmt.Sprintf("%02d:%02d", p.hour, p.minute)
	singleDigitHour = p.hourDigits == 1
	return date, tm, singleDigitHour
}
