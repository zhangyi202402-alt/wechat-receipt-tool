@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo [init] 创建今日门店目录...
wechat-receipt.exe init %*
if errorlevel 1 (
  echo 失败，请检查 config.yaml
  pause
  exit /b 1
)
echo 完成。请将截图放入 data\日期\门店名\ 目录
pause
