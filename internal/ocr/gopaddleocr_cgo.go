//go:build cgo

package ocr

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	gpaddle "github.com/multippt/gopaddleocr/pkg/ocr"

	"github.com/kalading/wechat-receipt-tool/internal/config"
	"github.com/kalading/wechat-receipt-tool/internal/pathutil"
)

type goPaddleEngine struct {
	engine *gpaddle.Engine
}

func newGoPaddleEngine() Engine { return &goPaddleEngine{} }

func (e *goPaddleEngine) Init(exeDir string, cfg config.OCRConfig) error {
	modelsDir := pathutil.Resolve(exeDir, cfg.ModelsDir)
	ortLib := pathutil.Resolve(exeDir, cfg.OnnxRuntimeLib)
	if runtime.GOOS != "windows" {
		switch runtime.GOOS {
		case "darwin":
			if cfg.OnnxRuntimeLib == "lib/onnxruntime.dll" || cfg.OnnxRuntimeLib == "onnxruntime.dll" {
				ortLib = pathutil.Resolve(exeDir, "lib/libonnxruntime.dylib")
			}
		default:
			if cfg.OnnxRuntimeLib == "lib/onnxruntime.dll" || cfg.OnnxRuntimeLib == "onnxruntime.dll" {
				ortLib = pathutil.Resolve(exeDir, "lib/libonnxruntime.so")
			}
		}
	} else {
		if err := prepareORTEnvironment(exeDir, cfg.OnnxRuntimeLib); err != nil {
			return err
		}
		resolved, err := resolveORTLibrary(exeDir, cfg.OnnxRuntimeLib)
		if err != nil {
			return err
		}
		ortLib = resolved
	}
	if err := os.Setenv("MODELS_DIR", modelsDir); err != nil {
		return err
	}
	if err := os.Setenv("ORT_LIB_PATH", ortLib); err != nil {
		return err
	}

	e.engine = gpaddle.NewEngine()
	if err := e.engine.Init(); err != nil {
		return fmt.Errorf("gopaddleocr init: %w", err)
	}
	return nil
}

func (e *goPaddleEngine) Close() error {
	if e.engine == nil {
		return nil
	}
	return e.engine.Close()
}

func (e *goPaddleEngine) Recognize(imgPath string) ([]TextBox, error) {
	data, err := os.ReadFile(imgPath)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	results, err := e.engine.RunOCR(data)
	if err != nil {
		return nil, fmt.Errorf("run ocr: %w", err)
	}
	return convertResults(results), nil
}

func convertResults(results []gpaddle.Result) []TextBox {
	var boxes []TextBox
	for _, r := range results {
		boxes = append(boxes, resultToBox(r))
		for _, child := range r.Children {
			boxes = append(boxes, resultToBox(child))
		}
	}
	return boxes
}

func resultToBox(r gpaddle.Result) TextBox {
	var box [4][2]float64
	for i := 0; i < 4 && i < len(r.Box); i++ {
		box[i][0] = float64(r.Box[i][0])
		box[i][1] = float64(r.Box[i][1])
	}
	return TextBox{Text: r.Text, Box: box, Score: r.Score}
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
