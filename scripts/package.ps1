#Requires -Version 5.1
# 打包为：exe + config.yaml + init.bat + process.bat + lib + models
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Dist = Join-Path $Root "dist\wechat-receipt-win64"
$Version = "1.0.0"
$ZipPath = Join-Path $Root "wechat-receipt-v$Version-win64.zip"

if (-not (Test-Path (Join-Path $Root "release\wechat-receipt.exe"))) {
    Write-Error "请先运行 scripts\build.bat 编译 release\wechat-receipt.exe"
}

& (Join-Path $Root "scripts\download-models.ps1")

if (Test-Path $Dist) { Remove-Item $Dist -Recurse -Force }
New-Item -ItemType Directory -Force -Path $Dist, (Join-Path $Dist "lib"), (Join-Path $Dist "models") | Out-Null

Copy-Item (Join-Path $Root "release\wechat-receipt.exe") $Dist
Copy-Item (Join-Path $Root "scripts\dist\config.windows.yaml") (Join-Path $Dist "config.yaml")
Copy-Item (Join-Path $Root "scripts\dist\init.bat") $Dist
Copy-Item (Join-Path $Root "scripts\dist\process.bat") $Dist
Copy-Item (Join-Path $Root "scripts\dist\使用说明.txt") $Dist
Copy-Item (Join-Path $Root "release\models\*") (Join-Path $Dist "models")
Copy-Item (Join-Path $Root "release\lib\onnxruntime.dll") (Join-Path $Dist "lib") -ErrorAction SilentlyContinue
if (-not (Test-Path (Join-Path $Dist "lib\onnxruntime.dll"))) {
    Write-Warning "lib\onnxruntime.dll 缺失，请确认 download-models.ps1 已成功"
}

if (Test-Path $ZipPath) { Remove-Item $ZipPath -Force }
Compress-Archive -Path $Dist -DestinationPath $ZipPath
Write-Host "已打包: $ZipPath"
Write-Host "目录内容:"
Get-ChildItem $Dist | Format-Table Name
