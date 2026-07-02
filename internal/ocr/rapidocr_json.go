package ocr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/pathutil"
)

// RapidOCRJSONEngine invokes RapidOCR-json as a subprocess (Windows fallback).
// See README for setup: https://github.com/RapidAI/RapidOCR
type RapidOCRJSONEngine struct {
	binPath string
}

type rapidOCRResponse struct {
	Results []struct {
		Box   [][2]float64 `json:"box"`
		Text  string       `json:"text"`
		Score float64      `json:"score"`
	} `json:"results"`
}

func (e *RapidOCRJSONEngine) Init(exeDir string, cfg config.OCRConfig) error {
	e.binPath = pathutil.Resolve(exeDir, cfg.RapidOCRJsonBin)
	if _, err := os.Stat(e.binPath); err != nil {
		return fmt.Errorf("rapidocr-json binary not found at %s: %w", e.binPath, err)
	}
	return nil
}

func (e *RapidOCRJSONEngine) Close() error { return nil }

func (e *RapidOCRJSONEngine) Recognize(imgPath string) ([]TextBox, error) {
	abs, err := filepath.Abs(imgPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(e.binPath, abs)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rapidocr-json: %w: %s", err, stderr.String())
	}
	var resp rapidOCRResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parse rapidocr-json output: %w", err)
	}
	var boxes []TextBox
	for _, r := range resp.Results {
		var box [4][2]float64
		for i := 0; i < 4 && i < len(r.Box); i++ {
			box[i] = r.Box[i]
		}
		boxes = append(boxes, TextBox{Text: r.Text, Box: box, Score: r.Score})
	}
	return boxes, nil
}
