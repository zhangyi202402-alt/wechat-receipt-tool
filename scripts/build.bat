@echo off
setlocal
cd /d %~dp0\..

if not exist release mkdir release

set CGO_ENABLED=1
if /I not "%OS%"=="Windows_NT" (
  set GOOS=windows
  set GOARCH=amd64
)

echo Building wechat-receipt.exe ...
go build -o release\wechat-receipt.exe .\cmd\wechat-receipt
if errorlevel 1 exit /b 1

echo Build OK: release\wechat-receipt.exe
echo Run scripts\download-models.ps1 to fetch OCR models before first use.
