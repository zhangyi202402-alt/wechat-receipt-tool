#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
echo "[process] 识别截图并生成合并 Excel..."
./wechat-receipt process "$@"
echo "完成。Excel 位于 data/日期/收款记录.xlsx"
