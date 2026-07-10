package process

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kalading/wechat-receipt-tool/internal/models"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
	"github.com/kalading/wechat-receipt-tool/internal/ocrdebug"
)

type ocrDebugSection struct {
	Store     string
	ImageName string
	Body      string
}

func writeOCRDebug(path, dateStr string, sections []ocrDebugSection) error {
	var b strings.Builder
	fmt.Fprintf(&b, "OCR 调试报告\n")
	fmt.Fprintf(&b, "日期: %s\n", dateStr)
	fmt.Fprintf(&b, "生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "说明: 由 --debug-ocr 生成，用于排查金额/来源识别问题\n")
	fmt.Fprintf(&b, "===\n\n")
	for _, sec := range sections {
		b.WriteString(sec.Body)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func buildOCRDebugSection(store, imageName string, boxes []ocr.TextBox, records []models.ReceiptRecord) string {
	var b strings.Builder
	ocrdebug.WriteImageSection(&b, store, imageName, boxes, records)
	return b.String()
}
