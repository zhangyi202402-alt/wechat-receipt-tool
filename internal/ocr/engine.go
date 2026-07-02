package ocr

import (
	"fmt"
	"strings"

	"github.com/kalading/wechat-receipt-tool/internal/config"
)

type TextBox struct {
	Text  string
	Box   [4][2]float64
	Score float64
}

type Engine interface {
	Init(exeDir string, cfg config.OCRConfig) error
	Close() error
	Recognize(imgPath string) ([]TextBox, error)
}

func NewEngine(provider string) (Engine, error) {
	switch strings.ToLower(provider) {
	case "gopaddleocr":
		return newGoPaddleEngine(), nil
	case "rapidocr-json":
		return &RapidOCRJSONEngine{}, nil
	default:
		return nil, fmt.Errorf("unsupported ocr provider: %s", provider)
	}
}

func centerY(box [4][2]float64) float64 {
	return (box[0][1] + box[1][1] + box[2][1] + box[3][1]) / 4
}

func centerX(box [4][2]float64) float64 {
	return (box[0][0] + box[1][0] + box[2][0] + box[3][0]) / 4
}

func boxBottom(box [4][2]float64) float64 {
	maxY := box[0][1]
	for i := 1; i < 4; i++ {
		if box[i][1] > maxY {
			maxY = box[i][1]
		}
	}
	return maxY
}

func boxTop(box [4][2]float64) float64 {
	minY := box[0][1]
	for i := 1; i < 4; i++ {
		if box[i][1] < minY {
			minY = box[i][1]
		}
	}
	return minY
}

// ModelsReady checks required ONNX model files exist.
func ModelsReady(modelsDir string) bool {
	return modelsReadyImpl(modelsDir)
}
