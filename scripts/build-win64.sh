#!/usr/bin/env bash
# Mac 上交叉编译 Windows x64（需 brew install mingw-w64）
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p release

MINGW_GCC=""
for candidate in x86_64-w64-mingw32-gcc x86_64-w64-mingw32-gcc-15 x86_64-w64-mingw32-gcc-14; do
  if command -v "$candidate" >/dev/null 2>&1; then
    MINGW_GCC="$candidate"
    break
  fi
done
if [[ -z "$MINGW_GCC" ]]; then
  echo "未找到 MinGW 交叉编译器。请先安装: brew install mingw-w64" >&2
  exit 1
fi

export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64
export CC="$MINGW_GCC"

echo "Cross-compiling wechat-receipt.exe (CC=$CC) ..."
GOTOOLCHAIN=auto go build -o release/wechat-receipt.exe ./cmd/wechat-receipt
file release/wechat-receipt.exe
echo "Build OK: release/wechat-receipt.exe"
