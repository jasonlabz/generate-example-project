# proto 代码生成脚本（PowerShell）
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot

function Log-Info($msg) { Write-Host "[$(Get-Date -Format 'HH:mm:ss')] INFO: $msg" }
function Log-Error($msg) { Write-Host "[$(Get-Date -Format 'HH:mm:ss')] ERROR: $msg"; exit 1 }

if (-not (Get-Command protoc -ErrorAction SilentlyContinue)) {
    Log-Error "protoc 不存在。Windows: choco install protoc 或从 github.com/protocolbuffers/protobuf/releases 下载"
}

# 先把 GOPATH\bin 加入 PATH，避免已安装的插件被误判为缺失
$env:PATH = "$env:PATH;$(go env GOPATH)\bin"

# 与提交的生成物保持一致：protoc-gen-go 版本跟随 go.mod 的 protobuf 版本
$protocGenGoVersion = $env:PROTOC_GEN_GO_VERSION
if (-not $protocGenGoVersion) {
    $protocGenGoVersion = (go list -m -f '{{.Version}}' google.golang.org/protobuf)
}
$protocGenGoGrpcVersion = $env:PROTOC_GEN_GO_GRPC_VERSION
if (-not $protocGenGoGrpcVersion) { $protocGenGoGrpcVersion = "v1.6.2" }

foreach ($tool in @(
    @{ Cmd = "protoc-gen-go"; Pkg = "google.golang.org/protobuf/cmd/protoc-gen-go"; Ver = $protocGenGoVersion },
    @{ Cmd = "protoc-gen-go-grpc"; Pkg = "google.golang.org/grpc/cmd/protoc-gen-go-grpc"; Ver = $protocGenGoGrpcVersion }
)) {
    if (-not (Get-Command $tool.Cmd -ErrorAction SilentlyContinue)) {
        if ($env:AUTO_INSTALL_TOOLS -ne "true") {
            Log-Error "$($tool.Cmd) 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true"
        }
        Log-Info "Installing $($tool.Pkg)@$($tool.Ver)..."
        go install "$($tool.Pkg)@$($tool.Ver)"
        if ($LASTEXITCODE -ne 0) { Log-Error "$($tool.Cmd) 安装失败" }
    }
}

Set-Location $ProjectRoot
# protoc 拒绝绝对路径，必须转为相对路径
$protos = Get-ChildItem -Path "api/proto" -Filter "*.proto" -Recurse | ForEach-Object { Resolve-Path -Relative $_.FullName }
if (-not $protos) { Log-Error "api/proto 下没有 .proto 文件" }

Log-Info "Generating protobuf code..."
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative @protos
if ($LASTEXITCODE -ne 0) { Log-Error "protoc 生成失败" }
Log-Info "Generation completed"
