#!/usr/bin/env pwsh

param()

# 配置参数
$GENTOL_CMD = if ($env:GENTOL_CMD) { $env:GENTOL_CMD } else { "gentol" }
$OUTPUT_DIR = if ($env:OUTPUT_DIR) { $env:OUTPUT_DIR } else { "." }
$TEMPLATE_DIR = if ($env:TEMPLATE_DIR) { $env:TEMPLATE_DIR } else { "./template" }
$DSN = ""
# 数据库配置 #TODO: 修改对应参数
# TODO: 数据库类型  "mysql|postgres|sqlserver|oracle|sqlite|dm"
$DB_TYPE = "postgres"
# TODO: 数据库host
$DB_HOST = "****************"
# TODO: 数据库port
$DB_PORT = "8530"
# TODO: 数据库 用户
$DB_USER = "postgres"
# TODO: 数据库 密码
$DB_PASS = "****************"
# TODO: 数据库 库名
$DB_NAME = "database"
# TODO: 数据库 模式
$DB_SCHEMA = ""
# TODO: 需要生成的表结构，不配置则为全部
$TABLES = if ($env:TABLES) { $env:TABLES } else { "" }

# 生成配置
$MODEL_DIR = if ($env:MODEL_DIR) { $env:MODEL_DIR } else { "dal/db/model" }
$DAO_DIR = if ($env:DAO_DIR) { $env:DAO_DIR } else { "dal/db/dao" }

# 功能开关
$ONLY_MODEL = if ($env:ONLY_MODEL) { [bool]::Parse($env:ONLY_MODEL) } else { $false }
$USE_SQL_NULLABLE = if ($env:USE_SQL_NULLABLE) { [bool]::Parse($env:USE_SQL_NULLABLE) } else { $false }
$RUN_GOFMT = if ($env:RUN_GOFMT) { [bool]::Parse($env:RUN_GOFMT) } else { $true }
$GEN_HOOK = if ($env:GEN_HOOK) { [bool]::Parse($env:GEN_HOOK) } else { $true }

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

# 构建 DSN（如果未直接提供）
function Build-Dsn {
    if ($DSN -ne "") {
        return
    }

    switch ($DB_TYPE) {
        "mysql" {
            $script:DSN = "${DB_USER}:${DB_PASS}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}?parseTime=True&loc=Local"
        }
        "postgres" {
            $script:DSN = "user=$DB_USER password=$DB_PASS host=$DB_HOST port=$DB_PORT dbname=$DB_NAME sslmode=disable TimeZone=Asia/Shanghai"
        }
        "sqlserver" {
            $script:DSN = "user id=$DB_USER;password=$DB_PASS;server=$DB_HOST;port=$DB_PORT;database=$DB_NAME;encrypt=disable"
        }
        "oracle" {
            $script:DSN = "${DB_USER}/${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
        }
        "sqlite" {
            $script:DSN = $DB_NAME
        }
        "dm" {
            $script:DSN = "dm://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}?schema=$DB_SCHEMA"
        }
        default {
            Write-ErrorLog "Unsupported database type: $DB_TYPE"
        }
    }
}

# 构建命令参数
function Build-Args {
    $argsList = @(
        "--db_type=$DB_TYPE"
        "--dsn=`"$DSN`""
        "--model=$MODEL_DIR"
        "--dao=$DAO_DIR"
    )

    if ($TABLES -ne "") {
        $argsList += "--table=$TABLES"
    }

    if ($DB_SCHEMA -ne "") {
        $argsList += "--schema=$DB_SCHEMA"
    }

    if ($ONLY_MODEL) {
        $argsList += "--only_model"
    }

    if ($USE_SQL_NULLABLE) {
        $argsList += "--use_sql_nullable"
    }

    if ($RUN_GOFMT) {
        $argsList += "--rungofmt"
    }

    if ($GEN_HOOK) {
        $argsList += "--gen_hook"
    }

    return $argsList
}

# 检查依赖
function Check-Gentol {
    if (Get-Command $GENTOL_CMD -ErrorAction SilentlyContinue) {
        return
    }

    Write-InfoLog "Installing gentol..."
    go install github.com/jasonlabz/gentol@master

    # 重新检查安装是否成功
    if (-not (Get-Command $GENTOL_CMD -ErrorAction SilentlyContinue)) {
        # 尝试在Go的bin目录中查找
        $goBinPath = Join-Path $env:GOPATH "bin" "gentol.exe"
        if (Test-Path $goBinPath) {
            $script:GENTOL_CMD = $goBinPath
        } else {
            # 如果Go在默认位置，尝试使用完整路径
            if ($env:GOPATH) {
                $goBinPath = Join-Path $env:GOPATH "bin" "gentol"
                if (Test-Path $goBinPath) {
                    $script:GENTOL_CMD = $goBinPath
                }
            }
        }
    }
}

# 主流程
function Main {
    Write-InfoLog "Starting code generation with gentol..."

    Check-Gentol
    Build-Dsn

    $args = Build-Args
    $command = "$GENTOL_CMD $args"

    Write-InfoLog "Running: $command"

    try {
        Invoke-Expression $command
        if ($LASTEXITCODE -ne 0) {
            Write-ErrorLog "Code generation failed with exit code: $LASTEXITCODE"
        }
    }
    catch {
        Write-ErrorLog "Code generation failed: $_"
    }

    Write-InfoLog "Code generation completed!"
}

# 设置错误处理
$ErrorActionPreference = "Stop"

# 执行主函数
Main