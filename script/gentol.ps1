#!/usr/bin/env pwsh

param(
    [Parameter(Position = 0)]
    [string]$Command,

    [Parameter(Position = 1)]
    [string]$SqlFile
)

$ErrorActionPreference = "Stop"

$GENTOL_CMD = if ($env:GENTOL_CMD) { $env:GENTOL_CMD } else { "gentol" }
$DSN = if ($env:DSN) { $env:DSN } else { "" }
$DB_TYPE = if ($env:DB_TYPE) { $env:DB_TYPE } else { "" }
$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } else { "" }
$DB_PASS = if ($env:DB_PASS) { $env:DB_PASS } else { "" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "" }
$DB_SCHEMA = if ($env:DB_SCHEMA) { $env:DB_SCHEMA } else { "" }
$DB_CONF = if ($env:DB_CONF) { $env:DB_CONF } else { "db.toml" }
$TABLES = if ($env:TABLES) { $env:TABLES } else { "" }

$MODEL_DIR = if ($env:MODEL_DIR) { $env:MODEL_DIR } else { "dal/db/model" }
$DAO_DIR = if ($env:DAO_DIR) { $env:DAO_DIR } else { "dal/db/dao" }
$ONLY_MODEL = if ($env:ONLY_MODEL) { [bool]::Parse($env:ONLY_MODEL) } else { $false }
$USE_SQL_NULLABLE = if ($env:USE_SQL_NULLABLE) { [bool]::Parse($env:USE_SQL_NULLABLE) } else { $false }
$RUN_GOFMT = if ($env:RUN_GOFMT) { [bool]::Parse($env:RUN_GOFMT) } else { $true }
$GEN_HOOK = if ($env:GEN_HOOK) { [bool]::Parse($env:GEN_HOOK) } else { $true }

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$ConfFile = Join-Path (Join-Path $ProjectRoot "conf") "application.yaml"
$TomlConfFile = Join-Path (Join-Path (Join-Path $ProjectRoot "conf") "db") $DB_CONF

# Writes a timestamped log message.
function Write-Log {
    param([string]$Message)
    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] $Message"
}

# Writes an informational log message.
function Write-InfoLog {
    param([string]$Message)
    Write-Log "INFO: $Message"
}

# Writes an error log message and exits.
function Write-ErrorLog {
    param([string]$Message)
    Write-Log "ERROR: $Message"
    exit 1
}

# Removes a whitespace-prefixed inline comment and trailing whitespace.
function Remove-InlineComment {
    param([string]$Value)

    return [regex]::Replace($Value, '\s+#.*$', '').Trim()
}

# Reads a top-level field from the datasource section in application.yaml.
function Read-YamlValue {
    param([string]$Key)

    if (-not (Test-Path $ConfFile)) {
        return ""
    }

    $pattern = '^\s+{0}:\s*"?([^"]*)"?' -f [regex]::Escape($Key)
    $inDatasource = $false
    foreach ($line in Get-Content $ConfFile) {
        if ($line -match '^datasource:') {
            $inDatasource = $true
            continue
        }
        if ($inDatasource -and $line -match '^[a-z]') {
            break
        }
        if ($inDatasource -and $line -match $pattern) {
            return Remove-InlineComment $Matches[1]
        }
    }
    return ""
}

# Reads a field from the first item in a nested connection section.
function Read-YamlSubValue {
    param([string]$Section, [string]$Key)

    if (-not (Test-Path $ConfFile)) {
        return ""
    }

    $pattern = '^\s+-*\s*{0}:\s*"?([^"]*)"?' -f [regex]::Escape($Key)
    $inSection = $false
    foreach ($line in Get-Content $ConfFile) {
        if ($line -match "^  ${Section}:") {
            $inSection = $true
            continue
        }
        if ($inSection -and $line -match '^[a-z]') {
            break
        }
        if ($inSection -and $line -match $pattern) {
            return Remove-InlineComment $Matches[1]
        }
    }
    return ""
}

# Reads a connection field from top-level, masters[0], then replicas[0].
function Read-YamlConnValue {
    param([string]$Key)

    $value = Read-YamlValue $Key
    if ($value) { return $value }
    $value = Read-YamlSubValue "masters" $Key
    if ($value) { return $value }
    return Read-YamlSubValue "replicas" $Key
}

# Fills database settings missing from the environment with application.yaml.
function Load-YamlConfig {
    if (-not (Test-Path $ConfFile)) {
        return $false
    }

    if (-not $script:DB_TYPE) { $script:DB_TYPE = Read-YamlValue "db_type" }
    if (-not $script:DB_HOST) { $script:DB_HOST = Read-YamlConnValue "host" }
    if (-not $script:DB_PORT) { $script:DB_PORT = Read-YamlConnValue "port" }
    if (-not $script:DB_USER) {
        $script:DB_USER = Read-YamlConnValue "username"
        if (-not $script:DB_USER) { $script:DB_USER = Read-YamlConnValue "user" }
    }
    if (-not $script:DB_PASS) { $script:DB_PASS = Read-YamlConnValue "password" }
    if (-not $script:DB_NAME) { $script:DB_NAME = Read-YamlConnValue "database" }

    Write-InfoLog "Used application.yaml to fill missing database settings"
    return $true
}

# Reads a single value from the configured TOML database file.
function Read-TomlValue {
    param([string]$Key)

    if (-not (Test-Path $TomlConfFile)) {
        return ""
    }

    $pattern = '^\s*{0}\s*=\s*"?([^"]*)"?' -f [regex]::Escape($Key)
    foreach ($line in Get-Content $TomlConfFile) {
        if ($line -match $pattern) {
            return Remove-InlineComment $Matches[1]
        }
    }
    return ""
}

# Fills database settings missing from the environment with conf/db/<DB_CONF>.
function Load-TomlConfig {
    if (-not (Test-Path $TomlConfFile)) {
        return $false
    }

    if (-not $script:DB_TYPE) { $script:DB_TYPE = Read-TomlValue "DBDriver" }
    if (-not $script:DB_HOST) { $script:DB_HOST = Read-TomlValue "Host" }
    if (-not $script:DB_PORT) { $script:DB_PORT = Read-TomlValue "Port" }
    if (-not $script:DB_USER) { $script:DB_USER = Read-TomlValue "Username" }
    if (-not $script:DB_PASS) { $script:DB_PASS = Read-TomlValue "Password" }
    if (-not $script:DB_NAME) { $script:DB_NAME = Read-TomlValue "DBName" }
    if (-not $script:DB_SCHEMA) { $script:DB_SCHEMA = Read-TomlValue "SchemaName" }

    Write-InfoLog "Used $TomlConfFile to fill missing database settings"
    return $true
}

# Validates the database settings required by the selected mode.
function Assert-DbConfig {
    if (-not $DB_TYPE) {
        Write-ErrorLog "DB_TYPE is required; set it as an environment variable"
    }
    if ($DSN) {
        return
    }

    switch ($DB_TYPE) {
        "sqlite" {
            if (-not $DB_NAME) { Write-ErrorLog "sqlite requires DB_NAME" }
        }
        "dm" {
            if (-not $DB_HOST -or -not $DB_PORT -or -not $DB_USER) {
                Write-ErrorLog "dm requires DB_HOST, DB_PORT, and DB_USER"
            }
        }
        { $_ -in "mysql", "postgres", "sqlserver", "oracle" } {
            if (-not $DB_HOST -or -not $DB_PORT -or -not $DB_USER -or -not $DB_NAME) {
                Write-ErrorLog "$DB_TYPE requires DB_HOST, DB_PORT, DB_USER, and DB_NAME"
            }
        }
        default {
            Write-ErrorLog "Unsupported database type: $DB_TYPE"
        }
    }
}

# Builds the code-generation DSN from individual environment variables.
function Build-Dsn {
    if ($DSN) {
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
    }
}

# Checks gentol and preserves the template's automatic installation behavior.
function Test-Gentol {
    if (Get-Command $GENTOL_CMD -ErrorAction SilentlyContinue) {
        return
    }

    Write-InfoLog "Installing gentol..."
    go install github.com/jasonlabz/gentol@master

    if (-not (Get-Command $GENTOL_CMD -ErrorAction SilentlyContinue)) {
        $goPath = if ($env:GOPATH) { $env:GOPATH } else { go env GOPATH }
        $goBin = Join-Path $goPath "bin" "gentol.exe"
        if (Test-Path $goBin) {
            $script:GENTOL_CMD = $goBin
        } else {
            Write-ErrorLog "gentol was not found; verify Go and GOPATH"
        }
    }
}

# Generates DAO and Model code.
function New-DatabaseCode {
    Assert-DbConfig
    Build-Dsn

    $gentolArgs = @(
        "--db_type=$DB_TYPE",
        "--dsn=$DSN",
        "--model=$MODEL_DIR",
        "--dao=$DAO_DIR"
    )
    if ($TABLES) { $gentolArgs += "--table=$TABLES" }
    if ($DB_SCHEMA) { $gentolArgs += "--schema=$DB_SCHEMA" }
    if ($ONLY_MODEL) { $gentolArgs += "--only_model" }
    if ($USE_SQL_NULLABLE) { $gentolArgs += "--use_sql_nullable" }
    if ($RUN_GOFMT) { $gentolArgs += "--rungofmt" }
    if ($GEN_HOOK) { $gentolArgs += "--gen_hook" }

    Write-InfoLog "Starting code generation with gentol..."
    Push-Location $ProjectRoot
    try {
        & $GENTOL_CMD @gentolArgs
        if ($LASTEXITCODE -ne 0) {
            Write-ErrorLog "Code generation failed with exit code: $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }
    Write-InfoLog "Code generation completed!"
}

# Executes the specified DDL file.
function Invoke-GentolDdl {
    param([string]$Path)

    if (-not (Test-Path $Path)) {
        Write-ErrorLog "SQL file does not exist: $Path"
    }
    Assert-DbConfig

    $gentolArgs = @("ddl", $Path, "--db_type=$DB_TYPE")
    if ($DSN) {
        $gentolArgs += "--dsn=$DSN"
    } else {
        $gentolArgs += @(
            "--host=$DB_HOST",
            "--port=$DB_PORT",
            "--username=$DB_USER",
            "--password=$DB_PASS",
            "--database=$DB_NAME"
        )
    }
    if ($DB_SCHEMA) { $gentolArgs += "--schema=$DB_SCHEMA" }

    Write-InfoLog "Executing DDL: $Path"
    & $GENTOL_CMD @gentolArgs
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorLog "DDL execution failed with exit code: $LASTEXITCODE"
    }
    Write-InfoLog "DDL execution completed!"
}

# Prints usage for the unified script entry point.
function Show-Usage {
    Write-Host "Usage:"
    Write-Host "  gentol.ps1                 Generate DAO/Model code"
    Write-Host "  gentol.ps1 ddl <sql-file>  Execute a DDL file"
    exit 1
}

# Selects DDL execution or code generation from the ddl subcommand.
function Main {
    if (-not (Load-TomlConfig)) {
        if (-not (Load-YamlConfig)) {
            Write-InfoLog "No database configuration file was found; using environment variables only"
        }
    }

    if (-not $Command) {
        Test-Gentol
        New-DatabaseCode
    } elseif ($Command -eq "ddl" -and $SqlFile) {
        Test-Gentol
        Invoke-GentolDdl $SqlFile
    } else {
        Show-Usage
    }
}

Main
