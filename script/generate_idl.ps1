#!/usr/bin/env pwsh

param(
    [string]$KitexCmd = "kitex"
)

# 配置变量
$BASE_MODULE = "github.com/jasonlabz/generate-example-project"
$ROOT_DIR = Get-Location
$IDL_DIR = "idl"
$CLIENT_DIR = "client/kitex"
$SERVER_DIR = "server/kitex"

# 颜色输出函数
$ESC = [char]27
$RED = "${ESC}[0;31m"
$GREEN = "${ESC}[0;32m"
$YELLOW = "${ESC}[1;33m"
$NC = "${ESC}[0m" # No Color

function Write-InfoLog {
    param([string]$Message)
    Write-Host "${GREEN}[INFO]${NC} $Message"
}

function Write-WarnLog {
    param([string]$Message)
    Write-Host "${YELLOW}[WARN]${NC} $Message"
}

function Write-ErrorLog {
    param([string]$Message)
    Write-Host "${RED}[ERROR]${NC} $Message"
    exit 1
}

# 检查 kitex 是否可用
function Check-Kitex {
    # 首先检查命令是否存在
    try {
        $cmd = Get-Command $KitexCmd -ErrorAction Stop
        $script:KitexCmd = $cmd.Source
    }
    catch {
        Write-InfoLog "Installing kitex..."
        try {
            go install github.com/cloudwego/kitex/tool/cmd/kitex@latest

            # 安装后重新检查
            $kitexPath = Join-Path $env:GOPATH "bin" "kitex.exe"
            if (Test-Path $kitexPath) {
                $script:KitexCmd = $kitexPath
            }
            else {
                $kitexPath = Join-Path $env:GOPATH "bin" "kitex"
                if (Test-Path $kitexPath) {
                    $script:KitexCmd = $kitexPath
                }
                else {
                    Write-ErrorLog "kitex command not found and installation failed"
                }
            }
        }
        catch {
            Write-ErrorLog "Failed to install kitex: $_"
        }
    }

    # 检查模块是否存在
    try {
        $moduleList = go list -m all 2>$null
        if ($moduleList -match "github.com/cloudwego/kitex") {
            Write-InfoLog "module kitex already exist, skipping go get ..."
        }
        else {
            Write-InfoLog "module kitex not exist，ready get it..."
            go get -u github.com/cloudwego/kitex@latest
        }
    }
    catch {
        Write-WarnLog "Failed to check kitex module: $_"
    }

    Write-InfoLog "Using kitex: $KitexCmd"
}

# 从文件路径提取服务名（不带后缀）
function Get-ServiceName {
    param([string]$FilePath)

    $filename = Split-Path $FilePath -Leaf
    $serviceName = [System.IO.Path]::GetFileNameWithoutExtension($filename)
    return $serviceName
}

# 通用生成函数
function Invoke-KitexGeneration {
    param(
        [string]$Type,
        [string]$Message,
        [string]$ExtraArgs,
        [string]$IdlFile,
        [string]$GenPath = "",
        [bool]$UseService = $false  # 新增：是否使用服务名参数
    )

    Write-InfoLog $Message

    $baseArgs = "-module $BASE_MODULE"
    if ($GenPath -ne "") {
        $baseArgs = "$baseArgs -gen-path $GenPath"
    }

    # 如果启用服务模式，从文件名生成服务名
    if ($UseService) {
        $serviceName = Get-ServiceName $IdlFile
        $baseArgs = "$baseArgs -service $serviceName"
        Write-InfoLog "Generating service: $serviceName"
    }

    if (-not (Test-Path $IdlFile -PathType Leaf)) {
        Write-WarnLog "IDL file not found: $IdlFile, skipping..."
        return $false
    }

    Write-InfoLog "Generating from: $IdlFile"

    # 构建完整的命令
    $fullArgs = @()
    $baseArgs -split ' ' | Where-Object { $_ -ne "" } | ForEach-Object { $fullArgs += $_ }
    if ($ExtraArgs -ne "") {
        $ExtraArgs -split ' ' | Where-Object { $_ -ne "" } | ForEach-Object { $fullArgs += $_ }
    }
    $fullArgs += $IdlFile

    Write-InfoLog "Run command: $KitexCmd $($fullArgs -join ' ')"

    try {
        & $KitexCmd @fullArgs
        if ($LASTEXITCODE -ne 0) {
            Write-ErrorLog "Failed to generate $Type from $IdlFile"
            return $false
        }
    }
    catch {
        Write-ErrorLog "Failed to generate $Type from $IdlFile, error: $_"
        return $false
    }

    Write-InfoLog "Successfully generated $Type from $IdlFile"
    return $true
}

# 遍历生成所有 thrift 文件（默认不加 -service）
function Generate-Thrift {
    Write-InfoLog "Scanning for thrift files in $IDL_DIR/client..."

    $clientIdlPath = Join-Path $IDL_DIR "client"
    if (-not (Test-Path $clientIdlPath -PathType Container)) {
        Write-WarnLog "IDL directory '$clientIdlPath' not found, skipping thrift generation"
        return $true
    }

    if (-not (Test-Path $CLIENT_DIR -PathType Container)) {
        New-Item -ItemType Directory -Path $CLIENT_DIR -Force | Out-Null
    }

    $thriftFiles = Get-ChildItem -Path $clientIdlPath -Filter "*.thrift" -File -Recurse | Select-Object -ExpandProperty FullName

    if ($thriftFiles.Count -eq 0) {
        Write-WarnLog "No thrift files found in $clientIdlPath"
        return $true
    }

    Write-InfoLog "Found $($thriftFiles.Count) thrift file(s):"
    foreach ($file in $thriftFiles) {
        Write-InfoLog "  - $file"
    }

    $successCount = 0
    $failCount = 0

    foreach ($thriftFile in $thriftFiles) {
        Write-InfoLog "Processing: $thriftFile"

        if (Invoke-KitexGeneration -Type "thrift" `
			-Message "Generate thrift from $thriftFile..." `
            -ExtraArgs "-thrift frugal_tag -invoker" `
			-IdlFile $thriftFile `
            -GenPath $CLIENT_DIR) {
            $successCount++
        }
        else {
            $failCount++
        }
    }

    Write-InfoLog "Thrift generation completed: $successCount successful, $failCount failed"
    return $failCount -eq 0
}

# 生成 thrift 服务端代码（新增：带 -service 参数，基于文件名）
function Generate-ThriftService {
    Write-InfoLog "Scanning for thrift files in $IDL_DIR/server (with service mode)..."

    $serverIdlPath = Join-Path $IDL_DIR "server"
    if (-not (Test-Path $serverIdlPath -PathType Container)) {
        Write-WarnLog "IDL directory '$serverIdlPath' not found, skipping thrift service generation"
        return $true
    }

    if (-not (Test-Path $SERVER_DIR -PathType Container)) {
        New-Item -ItemType Directory -Path $SERVER_DIR -Force | Out-Null
    }

    $thriftFiles = Get-ChildItem -Path $serverIdlPath -Filter "*.thrift" -File -Recurse | Select-Object -ExpandProperty FullName

    if ($thriftFiles.Count -eq 0) {
        Write-WarnLog "No thrift files found in $serverIdlPath for service generation"
        return $true
    }

    $successCount = 0
    $failCount = 0

    foreach ($thriftFile in $thriftFiles) {
        Write-InfoLog "Processing (service): $thriftFile"

        if (Invoke-KitexGeneration -Type "thrift service" `
			-Message "Generate thrift service from $thriftFile..." `
            -ExtraArgs "-thrift frugal_tag -invoker" `
			-IdlFile $thriftFile `
            -GenPath $SERVER_DIR `
			-UseService $true) {
			$successCount++
		}
		else {
			$failCount++
		}
	}

	Write-InfoLog "Thrift service generation completed: $successCount successful, $failCount failed"
	return $failCount -eq 0
}

# 遍历生成所有 slim thrift 文件（默认不加 -service）
function Generate-ThriftSlim {
    Write-InfoLog "Scanning for thrift files in $IDL_DIR/client (slim mode)..."

    $clientIdlPath = Join-Path $IDL_DIR "client"
    if (-not (Test-Path $clientIdlPath -PathType Container)) {
        Write-WarnLog "IDL directory '$clientIdlPath' not found, skipping slim thrift generation"
        return $true
    }

    if (-not (Test-Path $CLIENT_DIR -PathType Container)) {
        New-Item -ItemType Directory -Path $CLIENT_DIR -Force | Out-Null
    }

    $thriftFiles = Get-ChildItem -Path $clientIdlPath -Filter "*.thrift" -File -Recurse | Select-Object -ExpandProperty FullName

    if ($thriftFiles.Count -eq 0) {
        Write-WarnLog "No thrift files found in $clientIdlPath for slim generation"
        return $true
    }

    $successCount = 0
    $failCount = 0

    foreach ($thriftFile in $thriftFiles) {
        Write-InfoLog "Processing (slim): $thriftFile"

        if (Invoke-KitexGeneration -Type "thrift(slim)" `
			-Message "Generate slim thrift from $thriftFile..." `
            -ExtraArgs "-thrift frugal_tag -thrift template=slim" `
			-IdlFile $thriftFile `
            -GenPath "$CLIENT_DIR/slim") {
            $successCount++
        }
        else {
            $failCount++
        }
    }

    Write-InfoLog "Slim thrift generation completed: $successCount successful, $failCount failed"
    return $failCount -eq 0
}

# 生成 slim thrift 服务端代码（新增：带 -service 参数，基于文件名）
function Generate-ThriftSlimService {
    Write-InfoLog "Scanning for thrift files in $IDL_DIR/server (slim mode with service)..."

    $serverIdlPath = Join-Path $IDL_DIR "server"
    if (-not (Test-Path $serverIdlPath -PathType Container)) {
        Write-WarnLog "IDL directory '$serverIdlPath' not found, skipping slim thrift service generation"
        return $true
    }

    if (-not (Test-Path $SERVER_DIR -PathType Container)) {
        New-Item -ItemType Directory -Path $SERVER_DIR -Force | Out-Null
    }

    $thriftFiles = Get-ChildItem -Path $serverIdlPath -Filter "*.thrift" -File -Recurse | Select-Object -ExpandProperty FullName

    if ($thriftFiles.Count -eq 0) {
        Write-WarnLog "No thrift files found in $serverIdlPath for slim service generation"
        return $true
    }

    $successCount = 0
    $failCount = 0

    foreach ($thriftFile in $thriftFiles) {
        Write-InfoLog "Processing (slim service): $thriftFile"

        if (Invoke-KitexGeneration -Type "thrift(slim) service" `
			-Message "Generate slim thrift service from $thriftFile..." `
            -ExtraArgs "-thrift frugal_tag -thrift template=slim" `
			-IdlFile $thriftFile `
            -GenPath "$SERVER_DIR/slim" `
			-UseService $true) {
			$successCount++
		}
		else {
			$failCount++
		}
	}

	Write-InfoLog "Slim thrift service generation completed: $successCount successful, $failCount failed"
	return $failCount -eq 0
}

# Protobuf 生成函数（默认不加 -service）
function Generate-Protobuf {
    Write-InfoLog "Scanning for proto files in $IDL_DIR/client..."

    $clientIdlPath = Join-Path $IDL_DIR "client"
    if (-not (Test-Path $clientIdlPath -PathType Container)) {
        Write-WarnLog "IDL directory '$clientIdlPath' not found, skipping protobuf generation"
        return $true
    }

    if (-not (Test-Path $CLIENT_DIR -PathType Container)) {
        New-Item -ItemType Directory -Path $CLIENT_DIR -Force | Out-Null
    }

    $protoFiles = Get-ChildItem -Path $clientIdlPath -Filter "*.proto" -File -Recurse | Select-Object -ExpandProperty FullName

    if ($protoFiles.Count -eq 0) {
        Write-WarnLog "No proto files found in $clientIdlPath"
        return $true
    }

    Write-InfoLog "Found $($protoFiles.Count) proto file(s):"
    foreach ($file in $protoFiles) {
        Write-InfoLog "  - $file"
    }

    $successCount = 0
    $failCount = 0

    foreach ($protoFile in $protoFiles) {
        Write-InfoLog "Processing: $protoFile"

        if (Invoke-KitexGeneration -Type "protobuf" `
			-Message "Generate protobuf from $protoFile..." `
            -ExtraArgs "" `
			-IdlFile $protoFile `
            -GenPath $CLIENT_DIR) {
            $successCount++
        }
        else {
            $failCount++
        }
    }

    Write-InfoLog "Protobuf generation completed: $successCount successful, $failCount failed"
    return $failCount -eq 0
}

# 生成 protobuf 服务端代码（新增：带 -service 参数，基于文件名）
function Generate-ProtobufService {
    Write-InfoLog "Scanning for proto files in $IDL_DIR/server (with service mode)..."

    $serverIdlPath = Join-Path $IDL_DIR "server"
    if (-not (Test-Path $serverIdlPath -PathType Container)) {
        Write-WarnLog "IDL directory '$serverIdlPath' not found, skipping protobuf service generation"
        return $true
    }

    if (-not (Test-Path $SERVER_DIR -PathType Container)) {
        New-Item -ItemType Directory -Path $SERVER_DIR -Force | Out-Null
    }

    $protoFiles = Get-ChildItem -Path $serverIdlPath -Filter "*.proto" -File -Recurse | Select-Object -ExpandProperty FullName

    if ($protoFiles.Count -eq 0) {
        Write-WarnLog "No proto files found in $serverIdlPath for service generation"
        return $true
    }

    $successCount = 0
    $failCount = 0

    foreach ($protoFile in $protoFiles) {
        Write-InfoLog "Processing (service): $protoFile"

        if (Invoke-KitexGeneration -Type "protobuf service" `
			-Message "Generate protobuf service from $protoFile..." `
            -ExtraArgs "" `
	      	-IdlFile $protoFile `
            -GenPath $SERVER_DIR `
			-UseService $true) {
            $successCount++
        }
        else {
            $failCount++
        }
    }

    Write-InfoLog "Protobuf service generation completed: $successCount successful, $failCount failed"
    return $failCount -eq 0
}

# 子模块重新生成函数
function Regenerate-Submodule {
    param(
        [string]$Path,
        [string]$Module,
        [string]$ThriftFile,
        [bool]$UseService = $false
    )

    if (-not (Test-Path $Path -PathType Container)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }

    $originalDir = Get-Location
    Set-Location $Path

    Write-InfoLog "Executing kitex command in $Path with module: $Module and thrift file: $ThriftFile"

    if (-not (Test-Path $ThriftFile -PathType Leaf)) {
        Write-WarnLog "Thrift file not found: $ThriftFile in $Path, skipping..."
        Set-Location $originalDir
        return $true
    }

    # 构建 kitex 命令
    $kitexArgs = @("-module", $Module)
    if ($UseService) {
        $serviceName = Get-ServiceName $ThriftFile
        $kitexArgs += "-service"
        $kitexArgs += $serviceName
    }
    $kitexArgs += $ThriftFile

    # 执行 kitex 命令
    try {
        & $KitexCmd @kitexArgs
        if ($LASTEXITCODE -ne 0) {
            Write-ErrorLog "Failed to generate for $Path"
            Set-Location $originalDir
            return $false
        }
    }
    catch {
        Write-ErrorLog "Failed to generate for $Path, error: $_"
        Set-Location $originalDir
        return $false
    }

    # 更新依赖
    Write-InfoLog "Updating dependencies for $Path..."
    try {
        go get github.com/cloudwego/kitex@latest
        go mod tidy
    }
    catch {
        Write-WarnLog "Failed to update dependencies for $Path, error: $_"
    }

    Set-Location $originalDir
    Write-InfoLog "Successfully regenerated $Path"
    return $true
}

# 主执行函数
function Main {
    Write-InfoLog "Starting IDL code generation..."
    Check-Kitex

    # 生成主 IDL 文件（默认不加 -service）
    if (-not (Generate-Thrift)) {
        Write-WarnLog "Thrift generation had issues, continuing..."
    }

    # if (-not (Generate-ThriftSlim)) {
    #     Write-WarnLog "Thrift slim generation had issues, continuing..."
    # }

    if (-not (Generate-Protobuf)) {
        Write-WarnLog "Protobuf generation had issues, continuing..."
    }

    # 如果需要生成服务端代码，取消注释下面的调用：
    if (-not (Generate-ThriftService)) {
        Write-WarnLog "Thrift service generation had issues, continuing..."
    }

    # if (-not (Generate-ThriftSlimService)) {
    #     Write-WarnLog "Thrift slim service generation had issues, continuing..."
    # }

    if (-not (Generate-ProtobufService)) {
        Write-WarnLog "Protobuf service generation had issues, continuing..."
    }

    # 子模块配置数组
    $submodules = @(
        # TODO: 模板格式-> @($path, $module, $thriftFile, $useService)
        # 例如："basic/example_shop", "example_shop", "idl/item.thrift", $false           # 仅客户端
        # 例如："basic/example_shop", "example_shop", "idl/item.thrift", $true      # 服务端（基于文件名）
    )

    # 遍历所有子模块（仅在数组非空时执行）
    if ($submodules.Count -gt 0) {
        foreach ($submod in $submodules) {
            $path, $module, $thriftFile, $useService = $submod
            Regenerate-Submodule -Path $path -Module $module -ThriftFile $thriftFile -UseService $useService
        }
    }
    else {
        Write-InfoLog "No submodules configured, skipping submodule generation."
    }

    Write-InfoLog "All IDL code generation completed!"
}

# 设置错误处理
$ErrorActionPreference = "Stop"

# 执行主函数
Main