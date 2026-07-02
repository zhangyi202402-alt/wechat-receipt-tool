package folders

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kalading/wechat-receipt-tool/internal/config"
)

func InitDateStores(cfg *config.Config, baseDir string, date time.Time) ([]string, error) {
	dateDir := filepath.Join(baseDir, date.Format(cfg.DateFormat))
	var created []string
	for _, store := range cfg.Stores {
		storeDir := filepath.Join(dateDir, store)
		if err := os.MkdirAll(storeDir, 0o755); err != nil {
			return created, fmt.Errorf("create %s: %w", storeDir, err)
		}
		created = append(created, storeDir)
	}
	return created, nil
}

func StoreDir(baseDir, dateStr, store string) string {
	return filepath.Join(baseDir, dateStr, store)
}

func DateDir(baseDir, dateStr string) string {
	return filepath.Join(baseDir, dateStr)
}
