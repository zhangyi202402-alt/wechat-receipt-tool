// 诊断 OCR 原始输出与 BlockParser 聚类结果（开发用）
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
	"github.com/kalading/wechat-receipt-tool/internal/ocrdebug"
	"github.com/kalading/wechat-receipt-tool/internal/pathutil"
)

func main() {
	img := flag.String("image", "", "screenshot path")
	cfgPath := flag.String("config", "config.yaml", "config path")
	year := flag.Int("year", 2026, "fallback year for date parsing")
	store := flag.String("store", "诊断", "store name label")
	flag.Parse()
	if *img == "" {
		fmt.Fprintln(os.Stderr, "usage: ocr-debug -image path/to.png [-config config.yaml]")
		os.Exit(2)
	}

	exeDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cfg, err := config.Load(pathutil.Resolve(exeDir, *cfgPath))
	if err != nil {
		panic(err)
	}

	engine, err := ocr.NewEngine(cfg.OCR.Provider)
	if err != nil {
		panic(err)
	}
	if err := engine.Init(exeDir, cfg.OCR); err != nil {
		panic(err)
	}
	defer engine.Close()

	boxes, err := engine.Recognize(*img)
	if err != nil {
		panic(err)
	}

	imageName := filepath.Base(*img)
	records := ocrdebug.ParseRecords(boxes, *year, *store, imageName)
	ocrdebug.WriteImageSection(os.Stdout, *store, imageName, boxes, records)
}
