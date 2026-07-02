package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/folders"
	"github.com/kalading/wechat-receipt-tool/internal/ocr"
	"github.com/kalading/wechat-receipt-tool/internal/pathutil"
	"github.com/kalading/wechat-receipt-tool/internal/process"
)

var (
	configPath string
	dateFlag   string
	storeFlag  string
	forceFlag  bool
)

func main() {
	root := &cobra.Command{
		Use:   "wechat-receipt",
		Short: "微信零钱明细截图识别与 Excel 生成工具",
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "配置文件路径（默认 exe 目录下 config.yaml）")

	root.AddCommand(initCmd(), processCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadRuntime() (*config.Config, string, error) {
	exeDir, err := pathutil.ExeDir()
	if err != nil {
		return nil, "", err
	}
	cfgPath := pathutil.ResolveConfig(exeDir, configPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, "", fmt.Errorf("load config %s: %w", cfgPath, err)
	}
	return cfg, exeDir, nil
}

func parseDate(cfg *config.Config, flag string) (time.Time, error) {
	raw := flag
	if raw == "" {
		raw = time.Now().Format(cfg.DateFormat)
	}
	return time.Parse(cfg.DateFormat, raw)
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "创建当日（或指定日期）门店目录",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, exeDir, err := loadRuntime()
			if err != nil {
				return err
			}
			date, err := parseDate(cfg, dateFlag)
			if err != nil {
				return fmt.Errorf("invalid date: %w", err)
			}
			baseDir := pathutil.Resolve(exeDir, cfg.BaseDir)
			created, err := folders.InitDateStores(cfg, baseDir, date)
			if err != nil {
				return err
			}
			fmt.Printf("已创建/确认 %d 个门店目录（%s）:\n", len(created), date.Format(cfg.DateFormat))
			for _, d := range created {
				fmt.Printf("  %s\n", d)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dateFlag, "date", "", "日期 YYYY-MM-DD（默认今天）")
	return cmd
}

func processCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process",
		Short: "识别门店截图并生成 Excel",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, exeDir, err := loadRuntime()
			if err != nil {
				return err
			}
			date, err := parseDate(cfg, dateFlag)
			if err != nil {
				return fmt.Errorf("invalid date: %w", err)
			}
			if forceFlag {
				cfg.Process.Overwrite = true
			}

			engine, err := ocr.NewEngine(cfg.OCR.Provider)
			if err != nil {
				return err
			}
			if err := engine.Init(exeDir, cfg.OCR); err != nil {
				return fmt.Errorf("ocr init: %w", err)
			}
			defer engine.Close()

			svc := process.NewService(cfg, exeDir, engine)
			result, err := svc.Run(date, storeFlag, forceFlag)
			if err != nil {
				return err
			}
			for _, r := range result.Reports {
				fmt.Printf("\n[%s] 图片:%d 识别:%d", r.Store, r.ImageCount, r.RawRecords)
				if r.SkippedReason != "" {
					fmt.Printf(" — %s", r.SkippedReason)
				}
			}
			fmt.Printf("\n\n合计: 识别 %d 条 | 合并去重 %d 条\n", result.TotalRaw, result.TotalDedup)
			if result.ExcelWritten {
				fmt.Printf("Excel: %s\n", result.ExcelPath)
			}
			if result.ExcelSkipped {
				fmt.Println("Excel: 已跳过（使用 --force 覆盖）")
			}
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "错误: %s\n", e)
			}
			if len(result.Errors) > 0 {
				return fmt.Errorf("%d store(s) failed", len(result.Errors))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dateFlag, "date", "", "日期 YYYY-MM-DD（默认今天）")
	cmd.Flags().StringVar(&storeFlag, "store", "", "仅处理指定门店")
	cmd.Flags().BoolVar(&forceFlag, "force", false, "覆盖已有 Excel")
	return cmd
}
