package process

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/excel"
	"github.com/kalading/wechat-receipt-tool/internal/folders"
	"github.com/kalading/wechat-receipt-tool/internal/models"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
	"github.com/kalading/wechat-receipt-tool/internal/parser"
	"github.com/kalading/wechat-receipt-tool/internal/report"
)

type Service struct {
	cfg    *config.Config
	exeDir string
	engine ocr.Engine
}

func NewService(cfg *config.Config, exeDir string, engine ocr.Engine) *Service {
	return &Service{cfg: cfg, exeDir: exeDir, engine: engine}
}

type Result struct {
	Date           string
	Reports        []report.StoreReport
	TotalRaw       int
	TotalDedup     int
	ExcelPath      string
	ExcelWritten   bool
	ExcelSkipped   bool
	Errors         []string
}

func (s *Service) Run(date time.Time, storeFilter string, force bool) (*Result, error) {
	baseDir := filepath.Join(s.exeDir, s.cfg.BaseDir)
	dateStr := date.Format(s.cfg.DateFormat)
	dateDir := folders.DateDir(baseDir, dateStr)

	stores := s.cfg.Stores
	if storeFilter != "" {
		found := false
		for _, st := range s.cfg.Stores {
			if st == storeFilter {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("store %q not in config.stores", storeFilter)
		}
		stores = []string{storeFilter}
	}

	res := &Result{Date: dateStr}
	var merged []models.ReceiptRecord

	for _, store := range stores {
		storeDir := folders.StoreDir(baseDir, dateStr, store)
		if _, err := os.Stat(storeDir); os.IsNotExist(err) {
			res.Reports = append(res.Reports, report.StoreReport{
				Store:         store,
				Date:          dateStr,
				ProcessedAt:   time.Now(),
				SkippedReason: "目录不存在，已跳过",
			})
			continue
		}
		sr, recs, err := s.scanStore(storeDir, store, dateStr)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", store, err))
		}
		res.Reports = append(res.Reports, *sr)
		merged = append(merged, recs...)
	}

	res.TotalRaw = len(merged)
	deduped := parser.Deduplicate(merged)
	res.TotalDedup = len(deduped)

	excelPath := filepath.Join(dateDir, s.cfg.OutputFilename)
	reportPath := filepath.Join(dateDir, s.cfg.ReportFilename)
	res.ExcelPath = excelPath

	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		return res, fmt.Errorf("create date dir: %w", err)
	}

	overwrite := force || s.cfg.Process.Overwrite
	if _, err := os.Stat(excelPath); err == nil && !overwrite {
		res.ExcelSkipped = true
	} else if res.TotalDedup > 0 || res.TotalRaw > 0 {
		if err := excel.NewWriter().Write(excelPath, deduped); err != nil {
			return res, fmt.Errorf("write excel: %w", err)
		}
		res.ExcelWritten = true
	} else if overwrite {
		// 无记录时也生成空表（仅表头+合计）
		if err := excel.NewWriter().Write(excelPath, deduped); err != nil {
			return res, fmt.Errorf("write excel: %w", err)
		}
		res.ExcelWritten = true
	}

	if err := report.NewWriter().WriteDateSummary(reportPath, dateStr, res.Reports, res.TotalRaw, res.TotalDedup, res.ExcelWritten, res.ExcelSkipped); err != nil {
		return res, fmt.Errorf("write report: %w", err)
	}

	return res, nil
}

func (s *Service) scanStore(storeDir, store, dateStr string) (*report.StoreReport, []models.ReceiptRecord, error) {
	sr := &report.StoreReport{
		Store:       store,
		Date:        dateStr,
		ProcessedAt: time.Now(),
	}

	images, err := listImages(storeDir, s.cfg)
	if err != nil {
		return sr, nil, err
	}
	sr.ImageCount = len(images)
	if len(images) == 0 {
		sr.SkippedReason = "无图片"
		return sr, nil, nil
	}

	fallbackYear, _ := time.Parse(s.cfg.DateFormat, dateStr)
	var all []models.ReceiptRecord
	var mu sync.Mutex
	workers := s.cfg.OCR.Workers
	if workers > len(images) {
		workers = len(images)
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string, len(images))
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for imgPath := range jobs {
				imgResult := report.ImageResult{Filename: filepath.Base(imgPath)}
				boxes, ocrErr := s.engine.Recognize(imgPath)
				if ocrErr != nil {
					imgResult.Error = ocrErr.Error()
					mu.Lock()
					sr.Images = append(sr.Images, imgResult)
					mu.Unlock()
					continue
				}
				p := parser.NewBlockParser(parser.Options{
					FallbackYear: fallbackYear.Year(),
					StoreName:    store,
					SourceImage:  filepath.Base(imgPath),
				})
				recs := p.Parse(boxes)
				imgResult.RecordCount = len(recs)
				for _, r := range recs {
					if r.LowConfidence {
						imgResult.LowConfidence = append(imgResult.LowConfidence,
							fmt.Sprintf("%s +%.2f", r.Date, r.Amount))
					}
				}
				mu.Lock()
				all = append(all, recs...)
				sr.Images = append(sr.Images, imgResult)
				mu.Unlock()
			}
		}()
	}
	for _, img := range images {
		jobs <- img
	}
	close(jobs)
	wg.Wait()

	sort.Slice(sr.Images, func(i, j int) bool {
		return sr.Images[i].Filename < sr.Images[j].Filename
	})

	sr.RawRecords = len(all)
	storeDeduped := parser.Deduplicate(all)
	sr.DedupRecords = len(storeDeduped)
	return sr, all, nil
}

func listImages(dir string, cfg *config.Config) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var images []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if lower == strings.ToLower(cfg.OutputFilename) || lower == strings.ToLower(cfg.ReportFilename) {
			continue
		}
		if cfg.IsImage(name) {
			images = append(images, filepath.Join(dir, name))
		}
	}
	sort.Strings(images)
	return images, nil
}
