# Requires PowerShell 5+
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$ModelsDir = Join-Path $Root "release\models"
$LibDir = Join-Path $Root "release\lib"

New-Item -ItemType Directory -Force -Path $ModelsDir | Out-Null
New-Item -ItemType Directory -Force -Path $LibDir | Out-Null

$models = @{
    "PP-OCRv5_server_rec.onnx" = "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv5/rec/ch_PP-OCRv5_rec_server_infer.onnx"
    "PP-OCRv5_server_det.onnx" = "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv5/det/ch_PP-OCRv5_server_det.onnx"
    "PP-LCNet_x1_0_textline_ori.onnx" = "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.4.0/onnx/PP-OCRv4/cls/ch_ppocr_mobile_v2.0_cls_infer.onnx"
}

foreach ($entry in $models.GetEnumerator()) {
    $dest = Join-Path $ModelsDir $entry.Key
    if (-not (Test-Path $dest)) {
        Write-Host "Downloading $($entry.Key) ..."
        Invoke-WebRequest -Uri $entry.Value -OutFile $dest -UseBasicParsing
    } else {
        Write-Host "Exists: $($entry.Key)"
    }
}

$ortVersion = "1.24.1"
$ortZip = "onnxruntime-win-x64-$ortVersion.zip"
$ortUrl = "https://github.com/microsoft/onnxruntime/releases/download/v$ortVersion/$ortZip"
$ortZipPath = Join-Path $Root "release\$ortZip"

if (-not (Test-Path (Join-Path $LibDir "onnxruntime.dll"))) {
    Write-Host "Downloading ONNX Runtime $ortVersion ..."
    Invoke-WebRequest -Uri $ortUrl -OutFile $ortZipPath -UseBasicParsing
    Expand-Archive -Path $ortZipPath -DestinationPath (Join-Path $Root "release\_ort") -Force
    Copy-Item (Join-Path $Root "release\_ort\onnxruntime-win-x64-$ortVersion\lib\onnxruntime.dll") $LibDir
    Remove-Item $ortZipPath -Force -ErrorAction SilentlyContinue
    Remove-Item (Join-Path $Root "release\_ort") -Recurse -Force -ErrorAction SilentlyContinue
}

$cfgExample = Join-Path $Root "config.example.yaml"
$cfgDest = Join-Path $Root "release\config.yaml"
if (-not (Test-Path $cfgDest)) {
    Copy-Item $cfgExample $cfgDest
}

Write-Host "Models and ONNX Runtime ready under release/"
