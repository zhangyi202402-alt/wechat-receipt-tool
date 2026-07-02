#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p release

# Homebrew clang may target a missing SDK; use Apple toolchain for CGO.
export SDKROOT="${SDKROOT:-$(xcrun --show-sdk-path)}"
export CC="${CC:-/usr/bin/clang}"
export CGO_ENABLED=1

# Match native CPU (Apple Silicon=arm64, Intel=x86_64). Default Go may be amd64 under Rosetta.
case "$(uname -m)" in
  arm64) export GOARCH=arm64 ;;
  x86_64) export GOARCH=amd64 ;;
esac

GOTOOLCHAIN=auto go build -o release/wechat-receipt ./cmd/wechat-receipt
echo "Build OK: release/wechat-receipt ($(file -b release/wechat-receipt))"
echo "Run: bash scripts/download-models.sh  (first time only)"
