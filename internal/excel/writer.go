package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/xuri/excelize/v2"

	"github.com/kalading/wechat-receipt-tool/internal/models"
)

const (
	headerFill = "#1F4E79"
	noteFill   = "#F2F2F2"
	totalFill  = "#D9E2F3"
	reviewFill = "#FFF2CC"
	colCount   = 12
)

var headers = []string{
	"序号", "门店", "类型", "收支", "对方/摘要", "金额（元）",
	"日期", "时间", "状态", "置信度", "待核对原因", "原图片段",
}

type WriteOptions struct {
	EmbedReviewImages bool
	DateDir           string // 用于读取待核对裁剪图并嵌入
}

type Writer struct{}

func NewWriter() *Writer { return &Writer{} }

func (w *Writer) Write(path string, records []models.ReceiptRecord, opts WriteOptions) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	idx, _ := f.GetSheetIndex(sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Family: "微软雅黑"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{headerFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	noteStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "808080", Family: "微软雅黑", Italic: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{noteFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	amountStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    4,
	})
	reviewStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "微软雅黑"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{reviewFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	reviewAmountStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "微软雅黑"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{reviewFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    4,
	})
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: "微软雅黑"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{totalFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	totalAmountStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: "微软雅黑"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{totalFill}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    4,
	})

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	noteCell, _ := excelize.CoordinatesToCellName(1, 2)
	f.SetCellValue(sheet, noteCell, "注：门店=子目录名；对方=截图标题解析；金额含正负；待核对行请看「待核对原因」并对照「原图片段」确认")
	f.MergeCell(sheet, noteCell, mustCell(colCount, 2))
	f.SetCellStyle(sheet, noteCell, mustCell(colCount, 2), noteStyle)

	_ = f.SetColWidth(sheet, "L", "L", 28)

	startRow := 3
	for i, rec := range records {
		row := startRow + i
		ds, as := dataStyle, amountStyle
		if rec.Status == models.StatusNeedsReview {
			ds, as = reviewStyle, reviewAmountStyle
		}
		setRow(f, sheet, row, rec, ds, as)
		if opts.EmbedReviewImages && rec.Status == models.StatusNeedsReview && rec.SnippetRelPath != "" && opts.DateDir != "" {
			abs := filepath.Join(opts.DateDir, filepath.FromSlash(rec.SnippetRelPath))
			data, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			if err := f.AddPictureFromBytes(sheet, mustCell(12, row), &excelize.Picture{
				Extension: ".png",
				File:      data,
				Format: &excelize.GraphicOptions{
					ScaleX:          0.55,
					ScaleY:          0.55,
					LockAspectRatio: true,
					Positioning:     "oneCell",
				},
			}); err != nil {
				return fmt.Errorf("embed review image row %d: %w", row, err)
			}
			_ = f.SetRowHeight(sheet, row, 64)
		}
	}

	totalRow := startRow + len(records)
	incomeRow := totalRow + 1
	expenseRow := totalRow + 2
	if len(records) > 0 {
		firstData := startRow
		lastData := startRow + len(records) - 1
		f.SetCellValue(sheet, mustCell(1, totalRow), "合计(净额)")
		f.SetCellFormula(sheet, mustCell(6, totalRow),
			fmt.Sprintf("SUM(F%d:F%d)", firstData, lastData))
		f.SetCellValue(sheet, mustCell(1, incomeRow), "收入合计")
		f.SetCellFormula(sheet, mustCell(6, incomeRow),
			fmt.Sprintf(`SUMIF(F%d:F%d,">0")`, firstData, lastData))
		f.SetCellValue(sheet, mustCell(1, expenseRow), "支出合计")
		f.SetCellFormula(sheet, mustCell(6, expenseRow),
			fmt.Sprintf(`SUMIF(F%d:F%d,"<0")`, firstData, lastData))
	} else {
		f.SetCellValue(sheet, mustCell(1, totalRow), "合计(净额)")
		f.SetCellValue(sheet, mustCell(6, totalRow), 0)
		f.SetCellValue(sheet, mustCell(1, incomeRow), "收入合计")
		f.SetCellValue(sheet, mustCell(6, incomeRow), 0)
		f.SetCellValue(sheet, mustCell(1, expenseRow), "支出合计")
		f.SetCellValue(sheet, mustCell(6, expenseRow), 0)
	}
	for _, row := range []int{totalRow, incomeRow, expenseRow} {
		for col := 1; col <= colCount; col++ {
			cell := mustCell(col, row)
			if col == 6 {
				f.SetCellStyle(sheet, cell, cell, totalAmountStyle)
			} else {
				f.SetCellStyle(sheet, cell, cell, totalStyle)
			}
		}
	}

	f.SetColWidth(sheet, "A", "A", 8)
	f.SetColWidth(sheet, "B", "E", 16)
	f.SetColWidth(sheet, "F", "F", 12)
	f.SetColWidth(sheet, "G", "H", 12)
	f.SetColWidth(sheet, "I", "J", 10)
	f.SetColWidth(sheet, "K", "K", 28)
	f.SetColWidth(sheet, "L", "L", 28)
	f.SetActiveSheet(idx)

	return f.SaveAs(path)
}

func setRow(f *excelize.File, sheet string, row int, rec models.ReceiptRecord, dataStyle, amountStyle int) {
	conf := ""
	if rec.Confidence > 0 || rec.Status == models.StatusNeedsReview {
		conf = fmt.Sprintf("%.0f%%", rec.Confidence*100)
	}
	reasons := strings.Join(rec.ReviewReasons, "；")
	vals := []interface{}{
		rec.Serial,
		rec.Transferor,
		rec.Type.LabelCN(),
		rec.Direction,
		rec.Source,
		rec.Amount,
		rec.Date,
		rec.Time,
		rec.StatusLabelCN(),
		conf,
		reasons,
		"", // 原图片段：待核对行嵌入缩略图
	}
	for col, v := range vals {
		cell := mustCell(col+1, row)
		f.SetCellValue(sheet, cell, v)
		if col == 5 {
			f.SetCellStyle(sheet, cell, cell, amountStyle)
		} else {
			f.SetCellStyle(sheet, cell, cell, dataStyle)
		}
	}
}

func mustCell(col, row int) string {
	c, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		panic(err)
	}
	return c
}
