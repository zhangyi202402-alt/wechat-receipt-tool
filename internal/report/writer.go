package report

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type ImageResult struct {
	Filename      string
	RecordCount   int
	Error         string
	LowConfidence []string
}

type StoreReport struct {
	Store         string
	Date          string
	ProcessedAt   time.Time
	ImageCount    int
	RawRecords    int
	DedupRecords  int
	Images        []ImageResult
	SkippedReason string
	ExcelWritten  bool
	ExcelSkipped  bool
}

type Writer struct{}

func NewWriter() *Writer { return &Writer{} }

func (w *Writer) WriteDateSummary(path, dateStr string, stores []StoreReport, totalRaw, totalDedup int, excelWritten, excelSkipped bool) error {
	var b strings.Builder
	fmt.Fprintf(&b, "日期: %s\n", dateStr)
	if len(stores) > 0 {
		fmt.Fprintf(&b, "处理时间: %s\n", stores[0].ProcessedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(&b, "门店数: %d | 总图片: %d | 识别记录: %d | 合并去重后: %d\n",
		len(stores), sumImages(stores), totalRaw, totalDedup)
	if excelSkipped {
		b.WriteString("Excel: 已跳过（文件已存在，使用 --force 覆盖）\n")
	} else if excelWritten {
		b.WriteString("Excel: 已生成（所有门店合并）\n")
	}
	b.WriteString("===\n")
	for _, sr := range stores {
		fmt.Fprintf(&b, "\n【%s】\n", sr.Store)
		if sr.SkippedReason != "" {
			fmt.Fprintf(&b, "状态: %s\n", sr.SkippedReason)
			continue
		}
		fmt.Fprintf(&b, "图片数: %d | 识别: %d | 去重: %d\n", sr.ImageCount, sr.RawRecords, sr.DedupRecords)
		for _, img := range sr.Images {
			if img.Error != "" {
				fmt.Fprintf(&b, "  %s: 失败 - %s\n", img.Filename, img.Error)
				continue
			}
			line := fmt.Sprintf("  %s: %d 条", img.Filename, img.RecordCount)
			if len(img.LowConfidence) > 0 {
				line += fmt.Sprintf(" | 低置信度 %d 条", len(img.LowConfidence))
			}
			fmt.Fprintf(&b, "%s\n", line)
		}
		if len(sr.Images) == 0 && sr.SkippedReason == "" {
			b.WriteString("  无图片\n")
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sumImages(stores []StoreReport) int {
	n := 0
	for _, s := range stores {
		n += s.ImageCount
	}
	return n
}
