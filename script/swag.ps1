#!/usr/bin/env pwsh

param(
    [string]$Output,
    [string]$Main,
    [switch]$Format,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

# 配置
$SWAG_CMD = if ($env:SWAG_CMD) { $env:SWAG_CMD } else { "swag" }
$SWAG_VERSION = if ($env:SWAG_VERSION) { $env:SWAG_VERSION } else { "latest" }
$AUTO_INSTALL_TOOLS = if ($env:AUTO_INSTALL_TOOLS) { [bool]::Parse($env:AUTO_INSTALL_TOOLS) } else { $false }
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$SWAG_DIR = if ($env:SWAG_DIR) { $env:SWAG_DIR } else { Join-Path $ProjectRoot "bin" }
$PROJECT_DIR = if ($env:PROJECT_DIR) { $env:PROJECT_DIR } else { $ProjectRoot }
$SWAG_OUTPUT_DIR = if ($env:SWAG_OUTPUT_DIR) { $env:SWAG_OUTPUT_DIR } else { "docs/swagger" }
$SWAG_MAIN_FILE = if ($env:SWAG_MAIN_FILE) { $env:SWAG_MAIN_FILE } else { "cmd/server/main.go" }
$SWAG_PARSE_DEPTH = if ($env:SWAG_PARSE_DEPTH) { $env:SWAG_PARSE_DEPTH } else { "2" }
$RUN_SWAG_FMT = if ($env:RUN_SWAG_FMT) { [bool]::Parse($env:RUN_SWAG_FMT) } else { $false }

# 日志函数
function Write-Log {
    param([string]$Message)
    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] $Message"
}

function Write-InfoLog {
    param([string]$Message)
    Write-Log "INFO: $Message"
}

function Write-ErrorLog {
    param([string]$Message)
    Write-Log "ERROR: $Message"
    exit 1
}

# 输出目录必须位于项目内，避免脚本覆盖外部路径。
function Assert-OutputDir {
    if (-not $SWAG_OUTPUT_DIR) {
        Write-ErrorLog "SWAG_OUTPUT_DIR 不能为空"
    }

    $normalized = $SWAG_OUTPUT_DIR.Replace('\', '/')
    if ($normalized -match '^[A-Za-z]:/' -or $normalized.StartsWith('/')) {
        Write-ErrorLog "SWAG_OUTPUT_DIR 必须是项目内相对路径"
    }
    if ($normalized -match '\.\.') {
        Write-ErrorLog "SWAG_OUTPUT_DIR 不得包含 .."
    }
}

# 检查依赖
function Check-Swag {
    if (Get-Command $SWAG_CMD -ErrorAction SilentlyContinue) {
        return $true
    }

    # 检查指定目录下的 swag 可执行文件
    $swagExe = Join-Path $SWAG_DIR "swag.exe"
    if (Test-Path $swagExe) {
        $script:SWAG_CMD = $swagExe
        return $true
    }

    $swagBin = Join-Path $SWAG_DIR "swag"
    if (Test-Path $swagBin) {
        $script:SWAG_CMD = $swagBin
        return $true
    }

    if (-not $AUTO_INSTALL_TOOLS) {
        Write-ErrorLog "swag 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true；SWAG_VERSION 默认 latest"
    }
    if (-not $SWAG_VERSION) {
        Write-ErrorLog "SWAG_VERSION 不能为空"
    }

    Write-InfoLog "Installing swag@$SWAG_VERSION..."
    go install "github.com/swaggo/swag/cmd/swag@$SWAG_VERSION"

    if (-not (Get-Command $SWAG_CMD -ErrorAction SilentlyContinue)) {
        $goPath = if ($env:GOPATH) { $env:GOPATH } else { go env GOPATH }
        $goBinSwagExe = Join-Path $goPath "bin" "swag.exe"
        $goBinSwag = Join-Path $goPath "bin" "swag"

        if (Test-Path $goBinSwagExe) {
            $script:SWAG_CMD = $goBinSwagExe
            return $true
        }
        if (Test-Path $goBinSwag) {
            $script:SWAG_CMD = $goBinSwag
            return $true
        }

        Write-ErrorLog "swag 安装完成但当前 PATH 和 GOPATH/bin 中仍不可用"
    }
    return $true
}

# 运行 swag，通过参数数组传递避免 shell 字符串重解析。
function Run-Swag {
    Write-InfoLog "Running: swag $($args -join ' ') (in $PROJECT_DIR)"

    Push-Location $PROJECT_DIR
    try {
        & $SWAG_CMD @args
        if ($LASTEXITCODE -ne 0) {
            Write-ErrorLog "swag $($args -join ' ') failed with exit code: $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }

    Write-InfoLog "swag $($args -join ' ') completed"
}

# 显示用法
function Show-Usage {
    Write-Host "用法: $($MyInvocation.MyCommand.Name) [--output <项目内相对目录>] [--main <入口文件>] [--format]"
    exit 1
}

# 解析参数
function Parse-Args {
    if ($Help) {
        Show-Usage
    }
    if ($Output) {
        $script:SWAG_OUTPUT_DIR = $Output
    }
    if ($Main) {
        $script:SWAG_MAIN_FILE = $Main
    }
    if ($Format) {
        $script:RUN_SWAG_FMT = $true
    }
}

# 主流程
function Main {
    Parse-Args
    Assert-OutputDir
    Write-InfoLog "Starting swag documentation generation..."

    if (-not (Check-Swag)) {
        Write-ErrorLog "swag not found and installation failed"
    }

    if ($RUN_SWAG_FMT) {
        Run-Swag fmt
    }
    Run-Swag init --generalInfo $SWAG_MAIN_FILE --output $SWAG_OUTPUT_DIR --parseDependency --parseDepth $SWAG_PARSE_DEPTH

    Write-InfoLog "Documentation generation completed: $PROJECT_DIR/$SWAG_OUTPUT_DIR"
}

Main
