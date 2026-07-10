# 微信收款截图识别工具

按日期/门店目录组织微信「零钱明细」截图，OCR 识别「二维码收款」记录，生成 Excel 台账。

## 功能

- `init`：创建 `{data}/{日期}/{门店}/` 目录（门店列表可配置）
- `process`：扫描截图 → OCR → 生成 `收款记录.xlsx` 与 `处理报告.txt`

## 字段说明

| Excel 列 | 含义 |
|----------|------|
| 转账人 | 门店子目录名称 |
| 转账来源 | 截图中「二维码收款-来自…」的付款方 |

## 快速开始（Windows）

1. 解压 `wechat-receipt-v1.0.0-win64.zip`
2. **首次使用**：双击 `VC_redist.x64.exe` 安装 VC++ 运行库（仅需一次）
3. 双击或在命令行运行：

```bat
wechat-receipt.exe init
wechat-receipt.exe process
```

4. 将微信零钱明细截图放入 `data\{日期}\{门店}\` 后再次执行 `process`

### 常用参数

```bat
wechat-receipt.exe init --date 2026-07-02
wechat-receipt.exe process --date 2026-07-02 --store 北京世纪金源店 --force
```

- `--force`：覆盖已有 Excel
- `--config`：指定配置文件路径
- `--debug-ocr`：生成 `data/{日期}/ocr-debug.txt`，包含每张截图的 OCR 原始文字框与解析结果（排查金额识别问题）

### Windows OCR 报错排查

若出现 `Error loading ONNX shared library ... The specified module could not be found`：

1. 确认已安装包内 `VC_redist.x64.exe`（或 [Microsoft VC++ Redistributable x64](https://learn.microsoft.com/zh-cn/cpp/windows/latest-supported-vc-redist)）
2. 确认 `lib/` 下包含全部 `*.dll`（至少 `onnxruntime.dll` 与 `onnxruntime_providers_shared.dll`）
3. 确认 `config.yaml` 中 `onnxruntime_lib: lib/onnxruntime.dll`
4. 使用最新 GitHub Actions 产物重新下载（旧包可能只含单个 DLL）

## 目录结构

```
wechat-receipt.exe
config.yaml
lib/onnxruntime.dll
lib/onnxruntime_providers_shared.dll
models/*.onnx
data/
  2026-07-02/
    收款记录.xlsx      ← 所有门店合并为一个 Excel
    处理报告.txt
    北京世纪金源店/
      截图.png         ← 门店子目录只放截图
    成都七道堰店/
      ...
```

## 发布打包

交付包根目录：

```
wechat-receipt-win64/          （或 wechat-receipt-macos/）
  wechat-receipt.exe           主程序
  config.yaml                  配置
  init.bat                     创建门店目录
  process.bat                  识别并生成 Excel
  VC_redist.x64.exe            VC++ 运行库（Windows，首次安装）
  lib/                         OCR 运行库
  models/                      OCR 模型
  使用说明.txt
```

### Windows（推荐：GitHub Actions 自动编译）

代码推送到 [GitHub 仓库](https://github.com/zhangyi202402-alt/wechat-receipt-tool) 的 `main` 分支后，Actions 会自动编译并发布到 **Releases** 页。

1. 打开仓库 **[Releases](https://github.com/zhangyi202402-alt/wechat-receipt-tool/releases)**，下载 `wechat-receipt-v1.0.0-win64.zip`（最新 `v1.0.0`）
2. 若 Releases 尚未出现，可在 **Actions → Release** 工作流中查看是否成功；Artifacts 里也有同名 zip 备份（保留 90 天）
3. 解压后先运行 `VC_redist.x64.exe`（首次）→ `init.bat` → 放截图 → `process.bat`

也可在 Actions 页手动 **Run workflow** 触发编译与发布。

本地 Windows 编译：

```bat
scripts\build.bat
powershell -ExecutionPolicy Bypass -File scripts\package.ps1
```

### macOS

```bash
bash scripts/build.sh
bash scripts/package.sh
```

生成 `wechat-receipt-v1.0.0-macos.zip`，解压后运行 `./init.sh` → 放截图 → `./process.sh`。

## 从源码构建

要求：Go 1.25+、CGO、C 编译器

### Windows

```bat
scripts\build.bat
powershell -ExecutionPolicy Bypass -File scripts\download-models.ps1
```

### macOS

Mac **可以运行** `process`，但必须：

1. 用 **CGO** 重新编译（默认 `go build` 不带 CGO 时 OCR 不可用）
2. 下载 OCR 模型 + `libonnxruntime.dylib`
3. `config.yaml` 中 `onnxruntime_lib` 改为 `lib/libonnxruntime.dylib`

```bash
# 若 Homebrew clang 报 SDK 错误，脚本会使用 /usr/bin/clang
bash scripts/build.sh
bash scripts/download-models.sh   # 首次约 170MB
cd release
./wechat-receipt init --date 2026-07-02
./wechat-receipt process --date 2026-07-02 --store 北京世纪金源店 --force
```

若 GitHub 下载 ONNX Runtime 失败，可手动下载 [onnxruntime-osx-arm64](https://github.com/microsoft/onnxruntime/releases)（Apple 芯片）或 `osx-x64`（Intel），将 `lib/libonnxruntime.dylib` 放到 `release/lib/`。

开发机单元测试（无需 OCR 模型）：

```bash
CGO_ENABLED=0 go test ./...
```

## OCR 配置

`config.yaml` 中 `ocr.provider`：

- `gopaddleocr`（默认）：内置 ONNX OCR，免费离线；默认对截图**右侧金额列二次 OCR**（2 倍放大）以提升金额准确率
- `rapidocr-json`：调用外部 [RapidOCR-json](https://github.com/RapidAI/RapidOCR) 可执行文件

金额列二次识别可在 `config.yaml` 关闭或调参：

```yaml
ocr:
  amount_column_ocr: true      # 默认 true
  amount_column_start: 0.55    # 从 55% 宽度起裁剪右侧
  amount_column_scale: 2.0     # 放大倍数
```

## 限制

- 付款方姓名常为脱敏（如 `*李`），只能识别截图可见文字
- iPhone HEIC 截图请先转为 PNG/JPG
- 首次使用需下载 OCR 模型（约 80–120MB）

## 许可证

工具代码 MIT；OCR 模型来自 RapidOCR / PaddleOCR（Apache 2.0）。
