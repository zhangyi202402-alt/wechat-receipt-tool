#!/usr/bin/env bash
# Download OCR models and ONNX Runtime for macOS/Linux development.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODELS_DIR="$ROOT/release/models"
LIB_DIR="$ROOT/release/lib"
mkdir -p "$MODELS_DIR" "$LIB_DIR"

download() {
  local name="$1" url="$2"
  local dest="$MODELS_DIR/$name"
  if [[ -f "$dest" ]]; then
    echo "Exists: $name"
  else
    echo "Downloading $name ..."
    curl -L --fail -o "$dest" "$url"
  fi
}

download "PP-OCRv5_server_rec.onnx" \
  "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv5/rec/ch_PP-OCRv5_rec_server_infer.onnx"
download "PP-OCRv5_server_det.onnx" \
  "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv5/det/ch_PP-OCRv5_server_det.onnx"
download "PP-LCNet_x1_0_textline_ori.onnx" \
  "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv4/cls/ch_ppocr_mobile_v2.0_cls_infer.onnx"

ORT_VERSION="1.24.1"
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$OS-$ARCH" in
  Darwin-arm64) ORT_PKG="onnxruntime-osx-arm64-$ORT_VERSION.tgz" ;;
  Darwin-x86_64) ORT_PKG="onnxruntime-osx-x64-$ORT_VERSION.tgz" ;;
  Linux-x86_64) ORT_PKG="onnxruntime-linux-x64-$ORT_VERSION.tgz" ;;
  *)
    echo "Unsupported platform: $OS $ARCH — download ONNX Runtime manually into release/lib/"
    ORT_PKG=""
    ;;
esac

if [[ -n "$ORT_PKG" ]]; then
  ORT_LIB="$LIB_DIR/libonnxruntime.dylib"
  if [[ ! -f "$ORT_LIB" ]]; then
    echo "Downloading ONNX Runtime $ORT_VERSION ($ORT_PKG) ..."
    TMP="$ROOT/release/_ort.tgz"
    if ! curl --http1.1 -L --fail --retry 3 -o "$TMP" "https://github.com/microsoft/onnxruntime/releases/download/v$ORT_VERSION/$ORT_PKG"; then
      echo "GitHub download failed; trying module cache fallback ..."
      MOD_CACHE="$(go env GOMODCACHE 2>/dev/null || echo "$HOME/go/pkg/mod")"
      case "$OS-$ARCH" in
        Darwin-arm64)
          FALLBACK="$MOD_CACHE/github.com/yalue/onnxruntime_go@v1.27.0/test_data/onnxruntime_arm64.dylib"
          ;;
        Darwin-x86_64)
          FALLBACK="$MOD_CACHE/github.com/yalue/onnxruntime_go@v1.27.0/test_data/onnxruntime_arm64.dylib"
          ;;
        Linux-x86_64)
          FALLBACK="$MOD_CACHE/github.com/yalue/onnxruntime_go@v1.27.0/test_data/onnxruntime_arm64.so"
          ;;
      esac
      if [[ -f "$FALLBACK" ]]; then
        cp "$FALLBACK" "$ORT_LIB"
        echo "Copied from $FALLBACK"
      else
        echo "ERROR: could not download or find ONNX Runtime $ORT_VERSION for $OS-$ARCH"
        exit 1
      fi
    else
      tar -xzf "$TMP" -C "$ROOT/release"
      ORT_DIR="$ROOT/release/${ORT_PKG%.tgz}"
      cp "$ORT_DIR/lib/libonnxruntime."* "$LIB_DIR/" 2>/dev/null || cp "$ORT_DIR/lib/onnxruntime.so" "$LIB_DIR/" 2>/dev/null || true
      # Prefer stable symlink name for config
      if [[ -f "$LIB_DIR/libonnxruntime.1.24.1.dylib" ]]; then
        cp "$LIB_DIR/libonnxruntime.1.24.1.dylib" "$ORT_LIB"
      elif [[ -f "$LIB_DIR/libonnxruntime.so.1.24.1" ]]; then
        cp "$LIB_DIR/libonnxruntime.so.1.24.1" "$ORT_LIB"
      fi
      rm -rf "$TMP" "$ORT_DIR"
    fi
  else
    echo "Exists: libonnxruntime.dylib"
  fi
fi

CFG="$ROOT/release/config.yaml"
if [[ ! -f "$CFG" ]]; then
  cp "$ROOT/config.example.yaml" "$CFG"
fi

# Patch macOS config paths if needed
if [[ "$OS" == "Darwin" ]] && grep -q 'onnxruntime.dll' "$CFG" 2>/dev/null; then
  sed -i '' 's|onnxruntime_lib: lib/onnxruntime.dll|onnxruntime_lib: lib/libonnxruntime.dylib|' "$CFG" || true
fi

echo "Ready: release/{models,lib,config.yaml}"
