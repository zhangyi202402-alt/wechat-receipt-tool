#!/usr/bin/env bash
# 下载 Windows x64 OCR 模型与 ONNX Runtime（Mac 交叉编译用）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODELS_DIR="$ROOT/release/models"
LIB_DIR="$ROOT/release/lib"
mkdir -p "$MODELS_DIR" "$LIB_DIR"

download_model() {
  local name="$1" url="$2"
  local dest="$MODELS_DIR/$name"
  if [[ -f "$dest" ]]; then
    echo "Exists: $name"
  else
    echo "Downloading $name ..."
    curl -L --fail --retry 3 -o "$dest" "$url"
  fi
}

download_model "PP-OCRv5_server_rec.onnx" \
  "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv5/rec/ch_PP-OCRv5_rec_server_infer.onnx"
download_model "PP-OCRv5_server_det.onnx" \
  "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv5/det/ch_PP-OCRv5_server_det.onnx"
download_model "PP-LCNet_x1_0_textline_ori.onnx" \
  "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv4/cls/ch_ppocr_mobile_v2.0_cls_infer.onnx"

ORT_VERSION="1.24.1"
ORT_ZIP="onnxruntime-win-x64-${ORT_VERSION}.zip"
ORT_URL="https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/${ORT_ZIP}"

if [[ ! -f "$LIB_DIR/onnxruntime.dll" ]]; then
  echo "Downloading ONNX Runtime ${ORT_VERSION} (win-x64) ..."
  TMP_ZIP="$ROOT/release/$ORT_ZIP"
  curl -L --fail --retry 3 -o "$TMP_ZIP" "$ORT_URL"
  TMP_DIR="$ROOT/release/_ort"
  rm -rf "$TMP_DIR"
  unzip -q "$TMP_ZIP" -d "$TMP_DIR"
  cp "$TMP_DIR/onnxruntime-win-x64-${ORT_VERSION}/lib/"*.dll "$LIB_DIR/"
  rm -rf "$TMP_ZIP" "$TMP_DIR"
else
  echo "Exists: lib/onnxruntime.dll"
fi

if [[ ! -f "$LIB_DIR/onnxruntime.dll" ]]; then
  echo "error: onnxruntime.dll missing" >&2
  exit 1
fi

echo "Windows models and ONNX Runtime ready under release/"
