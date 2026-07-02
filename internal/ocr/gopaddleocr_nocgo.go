//go:build !cgo

package ocr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kalading/wechat-receipt-tool/internal/config"
)

type goPaddleEngine struct{}

func newGoPaddleEngine() Engine { return &goPaddleEngine{} }

func (e *goPaddleEngine) Init(_ string, _ config.OCRConfig) error {
	return fmt.Errorf("gopaddleocr requires CGO; build with CGO_ENABLED=1 on Windows, or set ocr.provider=rapidocr-json")
}

func (e *goPaddleEngine) Close() error { return nil }

func (e *goPaddleEngine) Recognize(_ string) ([]TextBox, error) {
	return nil, fmt.Errorf("gopaddleocr not available without CGO")
}

func modelsReadyImpl(modelsDir string) bool {
	required := []string{
		"PP-OCRv5_server_det.onnx",
		"PP-OCRv5_server_rec.onnx",
		"PP-LCNet_x1_0_textline_ori.onnx",
	}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(modelsDir, name)); err != nil {
			return false
		}
	}
	return true
}
