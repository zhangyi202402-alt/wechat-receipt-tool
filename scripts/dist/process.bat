@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo [process] 识别截图并生成合并 Excel...
wechat-receipt.exe process --force %*
if errorlevel 1 (
  echo 失败，请检查截图与 OCR 模型
  pause
  exit /b 1
)
echo 完成。Excel 位于 data\日期\收款记录.xlsx
pause
