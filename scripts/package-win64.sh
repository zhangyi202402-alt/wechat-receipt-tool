#!/usr/bin/env bash
# Mac 上打包 Windows x64 发布 zip
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist/wechat-receipt-win64"
VERSION="1.0.0"
ZIP_PATH="$ROOT/wechat-receipt-v${VERSION}-win64.zip"

if [[ ! -f "$ROOT/release/wechat-receipt.exe" ]]; then
  echo "请先运行: bash scripts/build-win64.sh" >&2
  exit 1
fi

bash "$ROOT/scripts/download-models-win.sh"

rm -rf "$DIST"
mkdir -p "$DIST/lib" "$DIST/models"

cp "$ROOT/release/wechat-receipt.exe" "$DIST/"
cp "$ROOT/scripts/dist/config.windows.yaml" "$DIST/config.yaml"
cp "$ROOT/scripts/dist/init.bat" "$DIST/"
cp "$ROOT/scripts/dist/process.bat" "$DIST/"
cp "$ROOT/scripts/dist/使用说明.txt" "$DIST/"
if [[ -f "$ROOT/scripts/dist/VC_redist.x64.exe" ]]; then
  cp "$ROOT/scripts/dist/VC_redist.x64.exe" "$DIST/"
else
  echo "警告: scripts/dist/VC_redist.x64.exe 未找到"
fi
cp "$ROOT/release/models/"* "$DIST/models/"
cp "$ROOT/release/lib/"*.dll "$DIST/lib/"

if [[ ! -f "$DIST/lib/onnxruntime.dll" ]]; then
  echo "error: lib/onnxruntime.dll missing" >&2
  exit 1
fi

rm -f "$ZIP_PATH"
(cd "$ROOT/dist" && zip -r "$ZIP_PATH" "wechat-receipt-win64")
echo "已打包: $ZIP_PATH"
ls -la "$DIST"
