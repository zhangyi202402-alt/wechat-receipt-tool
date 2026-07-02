#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
echo "[init] 创建今日门店目录..."
./wechat-receipt init "$@"
echo "完成。请将截图放入 data/日期/门店名/ 目录"
