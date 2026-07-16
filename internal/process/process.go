package process

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
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
	"github.com/kalading/wechat-receipt-tool/internal/review"
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
	Date         string
	Reports      []report.StoreReport
	TotalRaw     int
	TotalDedup   int
	ExcelPath    string
	ExcelWritten bool
	ExcelSkipped bool
	DebugOCRPath string
	Errors       []string
}

func (s *Service) Run(date time.Time, storeFilter string, force, debugOCR bool) (*Result, error) {
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
	var debugSections []ocrDebugSection

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
		sr, recs, dbg, err := s.scanStore(storeDir, store, dateStr, debugOCR)
		if debugOCR {
			debugSections = append(debugSections, dbg...)
		}
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", store, err))
		}
		res.Reports = append(res.Reports, *sr)
		merged = append(merged, recs...)
	}

	res.TotalRaw = len(merged)
	deduped := parser.Deduplicate(merged)

	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		return res, fmt.Errorf("create date dir: %w", err)
	}

	// 去重后再裁剪片段，保证序号与 Excel 一致，且路径不会因去重丢失
	saveSnippets := s.cfg.Process.SaveReviewSnippets == nil || *s.cfg.Process.SaveReviewSnippets
	if saveSnippets {
		deduped = attachReviewSnippets(deduped, baseDir, dateStr, dateDir)
	}
	res.TotalDedup = len(deduped)

	excelPath := filepath.Join(dateDir, s.cfg.OutputFilename)
	reportPath := filepath.Join(dateDir, s.cfg.ReportFilename)
	res.ExcelPath = excelPath

	overwrite := force || s.cfg.Process.Overwrite
	writeOpts := excel.WriteOptions{
		EmbedReviewImages: s.cfg.Process.EmbedReviewImages == nil || *s.cfg.Process.EmbedReviewImages,
		DateDir:           dateDir,
	}
	if _, err := os.Stat(excelPath); err == nil && !overwrite {
		res.ExcelSkipped = true
	} else if res.TotalDedup > 0 || res.TotalRaw > 0 {
		if err := excel.NewWriter().Write(excelPath, deduped, writeOpts); err != nil {
			return res, fmt.Errorf("write excel: %w", err)
		}
		res.ExcelWritten = true
	} else if overwrite {
		if err := excel.NewWriter().Write(excelPath, deduped, writeOpts); err != nil {
			return res, fmt.Errorf("write excel: %w", err)
		}
		res.ExcelWritten = true
	}

	// 图片已嵌入 Excel 后，删除临时 review 目录
	if res.ExcelWritten && writeOpts.EmbedReviewImages && saveSnippets {
		_ = os.RemoveAll(filepath.Join(dateDir, "review"))
	}

	// 报告用去重后的待核对统计
	for i := range res.Reports {
		n := 0
		for _, r := range deduped {
			if r.Transferor == res.Reports[i].Store && r.Status == models.StatusNeedsReview {
				n++
			}
		}
		res.Reports[i].NeedsReview = n
	}

	if err := report.NewWriter().WriteDateSummary(reportPath, dateStr, res.Reports, res.TotalRaw, res.TotalDedup, res.ExcelWritten, res.ExcelSkipped); err != nil {
		return res, fmt.Errorf("write report: %w", err)
	}

	if debugOCR && len(debugSections) > 0 {
		debugPath := filepath.Join(dateDir, "ocr-debug.txt")
		if err := writeOCRDebug(debugPath, dateStr, debugSections); err != nil {
			return res, fmt.Errorf("write ocr debug: %w", err)
		}
		res.DebugOCRPath = debugPath
	}

	return res, nil
}

func attachReviewSnippets(recs []models.ReceiptRecord, baseDir, dateStr, dateDir string) []models.ReceiptRecord {
	for i := range recs {
		if recs[i].Status != models.StatusNeedsReview {
			continue
		}
		if recs[i].SourceImage == "" || recs[i].Transferor == "" {
			continue
		}
		imgPath := filepath.Join(folders.StoreDir(baseDir, dateStr, recs[i].Transferor), recs[i].SourceImage)
		rel, err := review.SaveBandSnippet(
			imgPath, dateDir, recs[i].Transferor, recs[i].Serial, recs[i].Source, recs[i].BandBox,
		)
		if err != nil {
			// 再试一次：BandBox 退化为全宽，由 SaveBandSnippet 内部兜底
			rel, err = review.SaveBandSnippet(
				imgPath, dateDir, recs[i].Transferor, recs[i].Serial, recs[i].Source,
				[4][2]float64{},
			)
		}
		if err == nil {
			recs[i].SnippetRelPath = rel
		}
	}
	return recs
}

func (s *Service) scanStore(storeDir, store, dateStr string, debugOCR bool) (*report.StoreReport, []models.ReceiptRecord, []ocrDebugSection, error) {
	sr := &report.StoreReport{
		Store:       store,
		Date:        dateStr,
		ProcessedAt: time.Now(),
	}

	images, err := listImages(storeDir, s.cfg)
	if err != nil {
		return sr, nil, nil, err
	}
	sr.ImageCount = len(images)
	if len(images) == 0 {
		sr.SkippedReason = "无图片"
		return sr, nil, nil, nil
	}

	fallbackYear, _ := time.Parse(s.cfg.DateFormat, dateStr)
	var all []models.ReceiptRecord
	var debugSections []ocrDebugSection
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
				imgW := imageWidth(imgPath)
				p := parser.NewBlockParser(parser.Options{
					FallbackYear:          fallbackYear.Year(),
					StoreName:             store,
					SourceImage:           filepath.Base(imgPath),
					ImageWidth:            imgW,
					RequireDate:           s.cfg.Process.RequireDate,
					ReviewConfidenceBelow: s.cfg.Process.ReviewConfidenceBelow,
				})
				recs := filterTypes(p.Parse(boxes), s.cfg.Process.IncludeTypes)
				imgResult.RecordCount = len(recs)
				for _, r := range recs {
					if r.Status == models.StatusNeedsReview {
						imgResult.NeedsReview++
						reason := strings.Join(r.ReviewReasons, "；")
						imgResult.ReviewItems = append(imgResult.ReviewItems,
							fmt.Sprintf("%s %s %.2f [%s]", r.Date, r.Source, r.Amount, reason))
					}
					if r.LowConfidence {
						imgResult.LowConfidence = append(imgResult.LowConfidence,
							fmt.Sprintf("%s %s", r.Date, formatAmt(r.Amount)))
					}
				}
				if debugOCR {
					sec := buildOCRDebugSection(store, filepath.Base(imgPath), boxes, recs)
					mu.Lock()
					debugSections = append(debugSections, ocrDebugSection{
						Store: store, ImageName: filepath.Base(imgPath), Body: sec,
					})
					mu.Unlock()
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
	for _, r := range storeDeduped {
		if r.Status == models.StatusNeedsReview {
			sr.NeedsReview++
		}
	}
	return sr, all, debugSections, nil
}

func formatAmt(v float64) string {
	if v < 0 {
		return fmt.Sprintf("%.2f", v)
	}
	return fmt.Sprintf("+%.2f", v)
}

func imageWidth(path string) float64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0
	}
	return float64(cfg.Width)
}

func filterTypes(recs []models.ReceiptRecord, include []string) []models.ReceiptRecord {
	if len(include) == 0 {
		return recs
	}
	all := false
	allow := make(map[string]struct{})
	for _, t := range include {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "all" || t == "*" {
			all = true
			break
		}
		allow[t] = struct{}{}
	}
	if all {
		return recs
	}
	var out []models.ReceiptRecord
	for _, r := range recs {
		if _, ok := allow[string(r.Type)]; ok {
			out = append(out, r)
		}
	}
	return out
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
