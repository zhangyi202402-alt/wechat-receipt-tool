package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/kalading/wechat-receipt-tool/internal/models"
)

const (
	headerFill = "#1F4E79"
	noteFill   = "#F2F2F2"
	totalFill  = "#D9E2F3"
)

var headers = []string{"序号", "转账人", "转账来源", "转账金额（元）", "转账日期", "转账时间"}

type Writer struct{}

func NewWriter() *Writer { return &Writer{} }

func (w *Writer) Write(path string, records []models.ReceiptRecord) error {
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
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	amountStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "微软雅黑"},
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
	f.SetCellValue(sheet, noteCell, "注：转账人=门店名称，转账来源=截图付款方")
	f.MergeCell(sheet, noteCell, mustCell(6, 2))
	f.SetCellStyle(sheet, noteCell, mustCell(6, 2), noteStyle)

	startRow := 3
	for i, rec := range records {
		row := startRow + i
		setRow(f, sheet, row, rec, dataStyle, amountStyle)
	}

	totalRow := startRow + len(records)
	if len(records) > 0 {
		firstData := startRow
		lastData := startRow + len(records) - 1
		f.SetCellValue(sheet, mustCell(1, totalRow), "合计")
		f.SetCellFormula(sheet, mustCell(4, totalRow),
			fmt.Sprintf("SUM(D%d:D%d)", firstData, lastData))
	} else {
		f.SetCellValue(sheet, mustCell(1, totalRow), "合计")
		f.SetCellValue(sheet, mustCell(4, totalRow), 0)
	}
	for col := 1; col <= 6; col++ {
		cell := mustCell(col, totalRow)
		if col == 4 {
			f.SetCellStyle(sheet, cell, cell, totalAmountStyle)
		} else {
			f.SetCellStyle(sheet, cell, cell, totalStyle)
		}
	}

	f.SetColWidth(sheet, "A", "A", 8)
	f.SetColWidth(sheet, "B", "C", 18)
	f.SetColWidth(sheet, "D", "D", 16)
	f.SetColWidth(sheet, "E", "F", 14)
	f.SetActiveSheet(idx)

	return f.SaveAs(path)
}

func setRow(f *excelize.File, sheet string, row int, rec models.ReceiptRecord, dataStyle, amountStyle int) {
	vals := []interface{}{rec.Serial, rec.Transferor, rec.Source, rec.Amount, rec.Date, rec.Time}
	for col, v := range vals {
		cell := mustCell(col+1, row)
		f.SetCellValue(sheet, cell, v)
		if col == 3 {
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
