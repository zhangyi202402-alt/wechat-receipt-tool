package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type OCRConfig struct {
	Provider        string `yaml:"provider"`
	OnnxRuntimeLib  string `yaml:"onnxruntime_lib"`
	ModelsDir       string `yaml:"models_dir"`
	RapidOCRJsonBin string `yaml:"rapidocr_json_bin"`
	Workers         int    `yaml:"workers"`
}

type ProcessConfig struct {
	Overwrite bool `yaml:"overwrite"`
}

type Config struct {
	BaseDir          string        `yaml:"base_dir"`
	DateFormat       string        `yaml:"date_format"`
	Stores           []string      `yaml:"stores"`
	OutputFilename   string        `yaml:"output_filename"`
	ReportFilename   string        `yaml:"report_filename"`
	ImageExtensions  []string      `yaml:"image_extensions"`
	OCR              OCRConfig     `yaml:"ocr"`
	Process          ProcessConfig `yaml:"process"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.BaseDir == "" {
		c.BaseDir = "data"
	}
	if c.DateFormat == "" {
		c.DateFormat = "2006-01-02"
	}
	if c.OutputFilename == "" {
		c.OutputFilename = "收款记录.xlsx"
	}
	if c.ReportFilename == "" {
		c.ReportFilename = "处理报告.txt"
	}
	if len(c.ImageExtensions) == 0 {
		c.ImageExtensions = []string{".png", ".jpg", ".jpeg", ".webp"}
	}
	if c.OCR.Provider == "" {
		c.OCR.Provider = "gopaddleocr"
	}
	if c.OCR.OnnxRuntimeLib == "" {
		c.OCR.OnnxRuntimeLib = "lib/onnxruntime.dll"
	}
	if c.OCR.ModelsDir == "" {
		c.OCR.ModelsDir = "models"
	}
	if c.OCR.Workers <= 0 {
		c.OCR.Workers = 2
	}
	for i, ext := range c.ImageExtensions {
		if !strings.HasPrefix(ext, ".") {
			c.ImageExtensions[i] = "." + ext
		}
	}
}

func (c *Config) validate() error {
	if len(c.Stores) == 0 {
		return fmt.Errorf("config: stores must not be empty")
	}
	provider := strings.ToLower(c.OCR.Provider)
	if provider != "gopaddleocr" && provider != "rapidocr-json" {
		return fmt.Errorf("config: unsupported ocr.provider %q", c.OCR.Provider)
	}
	return nil
}

func (c *Config) IsImage(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range c.ImageExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func (c *Config) StoreSet() map[string]struct{} {
	set := make(map[string]struct{}, len(c.Stores))
	for _, s := range c.Stores {
		set[s] = struct{}{}
	}
	return set
}
