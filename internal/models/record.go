package models

import "time"

type ReceiptRecord struct {
	Serial        int
	Transferor    string // 转账人 = 门店名
	Source        string // 转账来源 = OCR 付款方
	Amount        float64
	Date          string // YYYY-MM-DD
	Time          string // HH:MM
	SourceImage   string
	ParsedAt      time.Time
	OCRScore      float64
	LowConfidence bool
}

type DedupKey struct {
	Transferor string
	Date       string
	Time       string
	Amount     float64
}

func (r ReceiptRecord) Key() DedupKey {
	return DedupKey{Transferor: r.Transferor, Date: r.Date, Time: r.Time, Amount: r.Amount}
}
