#!/usr/bin/env bash
# 打包为：wechat-receipt + config.yaml + init.sh + process.sh + lib + models
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="1.0.0"
DIST="$ROOT/dist/wechat-receipt-macos"
ZIP="$ROOT/wechat-receipt-v${VERSION}-macos.zip"

if [[ ! -f "$ROOT/release/wechat-receipt" ]]; then
  echo "请先运行: bash scripts/build.sh"
  exit 1
fi

bash "$ROOT/scripts/download-models.sh" || true

rm -rf "$DIST"
mkdir -p "$DIST/lib" "$DIST/models"

cp "$ROOT/release/wechat-receipt" "$DIST/"
cp "$ROOT/scripts/dist/config.mac.yaml" "$DIST/config.yaml"
cp "$ROOT/scripts/dist/init.sh" "$DIST/"
cp "$ROOT/scripts/dist/process.sh" "$DIST/"
cp "$ROOT/scripts/dist/使用说明.txt" "$DIST/"
chmod +x "$DIST/wechat-receipt" "$DIST/init.sh" "$DIST/process.sh"

cp "$ROOT/release/models/"*.onnx "$DIST/models/" 2>/dev/null || true
cp "$ROOT/release/models/"*.txt "$DIST/models/" 2>/dev/null || true
if [[ -f "$ROOT/release/lib/libonnxruntime.dylib" ]]; then
  cp "$ROOT/release/lib/libonnxruntime.dylib" "$DIST/lib/"
elif [[ -f "$ROOT/release/lib/libonnxruntime.1.20.1.dylib" ]]; then
  cp "$ROOT/release/lib/libonnxruntime.1.20.1.dylib" "$DIST/lib/libonnxruntime.dylib"
else
  MOD_CACHE="$(go env GOMODCACHE 2>/dev/null || echo "$HOME/go/pkg/mod")"
  FALLBACK="$MOD_CACHE/github.com/yalue/onnxruntime_go@v1.27.0/test_data/onnxruntime_arm64.dylib"
  if [[ -f "$FALLBACK" ]]; then
    cp "$FALLBACK" "$DIST/lib/libonnxruntime.dylib"
  else
    echo "警告: libonnxruntime.dylib 缺失，请先运行 download-models.sh"
  fi
fi

rm -f "$ZIP"
(cd "$ROOT/dist" && zip -r "$ZIP" "$(basename "$DIST")")
echo "已打包: $ZIP"
ls -la "$DIST"
