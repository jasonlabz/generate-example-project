# 模板改名脚本（PowerShell）
# 用法: ./script/rename.ps1 <new-service-name> <new-module-path>
param(
    [Parameter(Mandatory = $true)][string]$NewName,
    [Parameter(Mandatory = $true)][string]$NewModule
)

$ErrorActionPreference = "Stop"
$OldModule = "github.com/jasonlabz/generate-example-project"
$OldName = "generate-example-project"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

# 跳过 *.pb.go：生成代码内嵌带长度前缀的二进制描述符，纯文本替换会破坏长度字节
$patterns = @('*.go', '*.mod', '*.yaml', '*.yml', '*.sh', '*.ps1', '*.md', '*.proto', 'Makefile', 'Dockerfile')
$files = Get-ChildItem -Recurse -File -Include $patterns |
    Where-Object { $_.FullName -notmatch '\\\.git\\|\\bin\\|\\output\\' -and $_.Name -notlike '*.pb.go' }

foreach ($f in $files) {
    $content = Get-Content -Raw -Path $f.FullName
    # 先替换完整 module path，再替换裸服务名（module 含服务名子串，顺序不能反）
    $updated = $content.Replace($OldModule, $NewModule).Replace($OldName, $NewName)
    if ($updated -ne $content) {
        Set-Content -Path $f.FullName -Value $updated -NoNewline
    }
}

# 重新生成 pb 代码，刷新生成文件内嵌描述符中的 go_package 元数据
# （不重新生成也能正常编译运行，仅描述符元数据保留旧 module 字符串）
if (Get-Command protoc -ErrorAction SilentlyContinue) {
    try { & (Join-Path $PSScriptRoot 'proto.ps1') }
    catch { Write-Warning "pb 代码再生成失败，请稍后手动执行 script/proto.ps1" }
} else {
    Write-Host "提示: 未检测到 protoc，跳过 pb 代码再生成，请安装后执行 script/proto.ps1"
}

Write-Host "module 已替换为 $NewModule，服务名已替换为 $NewName"
Write-Host "后续: go mod tidy; make build"
