# proto 代码生成脚本（PowerShell）
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot

function Log-Info($msg) { Write-Host "[$(Get-Date -Format 'HH:mm:ss')] INFO: $msg" }
function Log-Error($msg) { Write-Host "[$(Get-Date -Format 'HH:mm:ss')] ERROR: $msg"; exit 1 }

if (-not (Get-Command protoc -ErrorAction SilentlyContinue)) {
    Log-Error "protoc 不存在。Windows: choco install protoc 或从 github.com/protocolbuffers/protobuf/releases 下载"
}

foreach ($tool in @(
    @{ Cmd = "protoc-gen-go"; Pkg = "google.golang.org/protobuf/cmd/protoc-gen-go" },
    @{ Cmd = "protoc-gen-go-grpc"; Pkg = "google.golang.org/grpc/cmd/protoc-gen-go-grpc" }
)) {
    if (-not (Get-Command $tool.Cmd -ErrorAction SilentlyContinue)) {
        if ($env:AUTO_INSTALL_TOOLS -ne "true") {
            Log-Error "$($tool.Cmd) 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true"
        }
        Log-Info "Installing $($tool.Pkg)@latest..."
        go install "$($tool.Pkg)@latest"
    }
}

$env:PATH = "$env:PATH;$(go env GOPATH)\bin"

Set-Location $ProjectRoot
$protos = Get-ChildItem -Path "api/proto" -Filter "*.proto" -Recurse | ForEach-Object { $_.FullName }
if (-not $protos) { Log-Error "api/proto 下没有 .proto 文件" }

Log-Info "Generating protobuf code..."
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative @protos
if ($LASTEXITCODE -ne 0) { Log-Error "protoc 生成失败" }
Log-Info "Generation completed"
