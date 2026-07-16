package models

import "time"

type TxType string

const (
	TxQRReceipt    TxType = "qr_receipt"
	TxTransferIn   TxType = "transfer_in"
	TxTransferOut  TxType = "transfer_out"
	TxRedPacket    TxType = "red_packet"
	TxRefund       TxType = "refund"
	TxWithdraw     TxType = "withdraw"
	TxMerchant     TxType = "merchant"
	TxOther        TxType = "other"
)

const (
	StatusOK          = "ok"
	StatusNeedsReview = "needs_review"
	DirectionIn       = "收入"
	DirectionOut      = "支出"
)

type ReceiptRecord struct {
	Serial         int
	Transferor     string // 门店名
	Source         string // 对方/标题摘要
	Amount         float64
	Date           string // YYYY-MM-DD
	Time           string // HH:MM
	SourceImage    string
	ParsedAt       time.Time
	OCRScore       float64
	LowConfidence  bool // 兼容旧报告：等同 Status==needs_review
	Type           TxType
	Direction      string // 收入 | 支出
	Title          string
	Status         string // ok | needs_review
	Missing        []string
	ReviewReasons  []string
	Confidence     float64
	SnippetRelPath string
	BandBox        [4][2]float64
}

type DedupKey struct {
	Transferor string
	Date       string
	Time       string
	Amount     float64
	Type       TxType
}

func (r ReceiptRecord) Key() DedupKey {
	return DedupKey{
		Transferor: r.Transferor,
		Date:       r.Date,
		Time:       r.Time,
		Amount:     r.Amount,
		Type:       r.Type,
	}
}

func (t TxType) LabelCN() string {
	switch t {
	case TxQRReceipt:
		return "二维码收款"
	case TxTransferIn:
		return "转账收入"
	case TxTransferOut:
		return "转账支出"
	case TxRedPacket:
		return "红包"
	case TxRefund:
		return "退款"
	case TxWithdraw:
		return "提现"
	case TxMerchant:
		return "商户消费"
	default:
		return "其他"
	}
}

func (r ReceiptRecord) StatusLabelCN() string {
	if r.Status == StatusNeedsReview {
		return "待核对"
	}
	return "正常"
}
