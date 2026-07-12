#!/bin/bash

set -euo pipefail

GENTOL_CMD="${GENTOL_CMD:-gentol}"
GENTOL_VERSION="${GENTOL_VERSION:-master}"
AUTO_INSTALL_TOOLS="${AUTO_INSTALL_TOOLS:-false}"
DSN="${DSN:-}"
DB_TYPE="${DB_TYPE:-}"
DB_HOST="${DB_HOST:-}"
DB_PORT="${DB_PORT:-}"
DB_USER="${DB_USER:-}"
DB_PASS="${DB_PASS:-}"
DB_NAME="${DB_NAME:-}"
DB_SCHEMA="${DB_SCHEMA:-}"
DB_CONF="${DB_CONF:-db.toml}"
TABLES="${TABLES:-}"

MODEL_DIR="${MODEL_DIR:-dal/db/model}"
DAO_DIR="${DAO_DIR:-dal/db/dao}"
ONLY_MODEL="${ONLY_MODEL:-false}"
USE_SQL_NULLABLE="${USE_SQL_NULLABLE:-false}"
RUN_GOFMT="${RUN_GOFMT:-true}"
GEN_HOOK="${GEN_HOOK:-true}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONF_FILE="$PROJECT_ROOT/conf/application.yaml"
TOML_CONF_FILE="$PROJECT_ROOT/conf/db/$DB_CONF"

log() { echo "[$(date '+%H:%M:%S')] $1"; }
log_info() { log "INFO: $1"; }
log_error() { log "ERROR: $1"; exit 1; }

# 清除值末尾以空白分隔的注释和多余空白。
clean_config_value() {
    printf '%s\n' "$1" | sed 's/[[:space:]][[:space:]]*#.*$//; s/[[:space:]]*$//'
}

# 从 application.yaml 读取 datasource 下的单个字段值。
yaml_val() {
    local value
    value=$(sed -n '/^datasource:/,/^[a-z]/p' "$CONF_FILE" 2>/dev/null | \
        sed -n "s/^  ${1}: *\"\?\([^\"]*\)\"\?/\1/p" | head -1)
    clean_config_value "$value"
}

# 读取嵌套连接块（masters/replicas）中首个 item 的字段值。
yaml_sub_val() {
    local section="$1"
    local key="$2"
    local value
    value=$(sed -n "/^  ${section}:/,/^[a-z]/p" "$CONF_FILE" 2>/dev/null | \
        sed -n "s/^[[:space:]]*-* *${key}: *\"\?\([^\"]*\)\"\?/\1/p" | head -1)
    clean_config_value "$value"
}

# 按 top-level、masters[0]、replicas[0] 的顺序读取连接字段。
yaml_conn_val() {
    local key="$1"
    local val
    val=$(yaml_val "$key")
    [[ -n "$val" ]] && echo "$val" && return
    val=$(yaml_sub_val "masters" "$key")
    [[ -n "$val" ]] && echo "$val" && return
    yaml_sub_val "replicas" "$key"
}

# 使用 application.yaml 补齐外部环境变量中未设置的数据库配置。
load_yaml_config() {
    if [[ ! -f "$CONF_FILE" ]]; then
        return 1
    fi

    [[ -z "$DB_TYPE" ]] && DB_TYPE=$(yaml_val "db_type")
    [[ -z "$DB_HOST" ]] && DB_HOST=$(yaml_conn_val "host")
    [[ -z "$DB_PORT" ]] && DB_PORT=$(yaml_conn_val "port")
    if [[ -z "$DB_USER" ]]; then
        DB_USER=$(yaml_conn_val "username")
        [[ -z "$DB_USER" ]] && DB_USER=$(yaml_conn_val "user")
    fi
    [[ -z "$DB_PASS" ]] && DB_PASS=$(yaml_conn_val "password")
    [[ -z "$DB_NAME" ]] && DB_NAME=$(yaml_conn_val "database")
    [[ -z "$DB_SCHEMA" ]] && DB_SCHEMA=$(yaml_conn_val "schema")

    log_info "已使用 application.yaml 补齐未设置的数据库配置"
}

# 从 conf/db/<DB_CONF> 读取并补齐未设置的数据库配置。
toml_val() {
    local value
    value=$(sed -n "s/^[[:space:]]*${1}[[:space:]]*=[[:space:]]*\"\?\([^\"]*\)\"\?.*/\1/p" "$TOML_CONF_FILE" 2>/dev/null | head -1)
    clean_config_value "$value"
}

# 使用 TOML 配置补齐外部环境变量中未设置的数据库配置。
load_toml_config() {
    if [[ ! -f "$TOML_CONF_FILE" ]]; then
        return 1
    fi

    [[ -z "$DB_TYPE" ]] && DB_TYPE=$(toml_val "DBDriver")
    [[ -z "$DB_HOST" ]] && DB_HOST=$(toml_val "Host")
    [[ -z "$DB_PORT" ]] && DB_PORT=$(toml_val "Port")
    [[ -z "$DB_USER" ]] && DB_USER=$(toml_val "Username")
    [[ -z "$DB_PASS" ]] && DB_PASS=$(toml_val "Password")
    [[ -z "$DB_NAME" ]] && DB_NAME=$(toml_val "DBName")
    [[ -z "$DB_SCHEMA" ]] && DB_SCHEMA=$(toml_val "SchemaName")

    log_info "已使用 $TOML_CONF_FILE 补齐未设置的数据库配置"
}

# 检查当前模式构建数据库连接所需的配置。
validate_db_config() {
    [[ -n "$DB_TYPE" ]] || log_error "缺少 DB_TYPE，请在脚本外设置环境变量"
    [[ -n "$DSN" ]] && return

    case "$DB_TYPE" in
        "sqlite")
            [[ -n "$DB_NAME" ]] || log_error "sqlite 需要设置 DB_NAME"
            ;;
        "dm")
            [[ -n "$DB_HOST" && -n "$DB_PORT" && -n "$DB_USER" ]] || \
                log_error "dm 需要设置 DB_HOST、DB_PORT 和 DB_USER"
            ;;
        "mysql"|"postgres"|"sqlserver"|"oracle")
            [[ -n "$DB_HOST" && -n "$DB_PORT" && -n "$DB_USER" && -n "$DB_NAME" ]] || \
                log_error "$DB_TYPE 需要设置 DB_HOST、DB_PORT、DB_USER 和 DB_NAME"
            ;;
        *)
            log_error "Unsupported database type: $DB_TYPE"
            ;;
    esac
}

# 根据分项环境变量构建代码生成使用的 DSN。
build_dsn() {
    [[ -n "$DSN" ]] && return

    case "$DB_TYPE" in
        "mysql")
            DSN="$DB_USER:$DB_PASS@tcp($DB_HOST:$DB_PORT)/$DB_NAME?parseTime=True&loc=Local"
            ;;
        "postgres")
            DSN="user=$DB_USER password=$DB_PASS host=$DB_HOST port=$DB_PORT dbname=$DB_NAME sslmode=disable TimeZone=Asia/Shanghai"
            ;;
        "sqlserver")
            DSN="user id=$DB_USER;password=$DB_PASS;server=$DB_HOST;port=$DB_PORT;database=$DB_NAME;encrypt=disable"
            ;;
        "oracle")
            DSN="$DB_USER/$DB_PASS@$DB_HOST:$DB_PORT/$DB_NAME"
            ;;
        "sqlite")
            DSN="$DB_NAME"
            ;;
        "dm")
            DSN="dm://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT?schema=$DB_SCHEMA"
            ;;
    esac
}

# 检查 gentol 命令；安装仍需显式允许，并记录所用版本引用。
check_gentol() {
    if command -v "$GENTOL_CMD" &>/dev/null; then
        return
    fi

    [[ "$AUTO_INSTALL_TOOLS" == "true" ]] || \
        log_error "gentol 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true；GENTOL_VERSION 默认 master"
    [[ -n "$GENTOL_VERSION" ]] || log_error "GENTOL_VERSION 不能为空"

    log_info "Installing gentol@$GENTOL_VERSION..."
    go install "github.com/jasonlabz/gentol@$GENTOL_VERSION"
    if ! command -v "$GENTOL_CMD" &>/dev/null; then
        local go_bin
        go_bin="$(go env GOPATH)/bin/gentol"
        [[ -x "$go_bin" ]] || log_error "gentol 安装完成但当前 PATH 和 GOPATH/bin 中仍不可用"
        GENTOL_CMD="$go_bin"
    fi
}

# 生成 DAO 和 Model 代码。
generate_code() {
    validate_db_config
    build_dsn

    local args=(
        "--db_type=$DB_TYPE"
        "--dsn=$DSN"
        "--model=$MODEL_DIR"
        "--dao=$DAO_DIR"
    )
    [[ -n "$TABLES" ]] && args+=("--table=$TABLES")
    [[ -n "$DB_SCHEMA" ]] && args+=("--schema=$DB_SCHEMA")
    [[ "$ONLY_MODEL" == "true" ]] && args+=("--only_model")
    [[ "$USE_SQL_NULLABLE" == "true" ]] && args+=("--use_sql_nullable")
    [[ "$RUN_GOFMT" == "true" ]] && args+=("--rungofmt")
    [[ "$GEN_HOOK" == "true" ]] && args+=("--gen_hook")

    log_info "Starting code generation with gentol..."
    if ! (cd "$PROJECT_ROOT" && "$GENTOL_CMD" "${args[@]}"); then
        log_error "Code generation failed"
    fi
    log_info "Code generation completed!"
}

# 执行指定 DDL 文件。
execute_ddl() {
    local sql_file="$1"
    [[ "$sql_file" == /* ]] || sql_file="$PROJECT_ROOT/$sql_file"
    [[ -f "$sql_file" ]] || log_error "SQL文件不存在: $sql_file"

    validate_db_config

    local args=("ddl" "$sql_file" "--db_type=$DB_TYPE")
    if [[ -n "$DSN" ]]; then
        args+=("--dsn=$DSN")
    else
        args+=(
            "--host=$DB_HOST"
            "--port=$DB_PORT"
            "--username=$DB_USER"
            "--password=$DB_PASS"
            "--database=$DB_NAME"
        )
    fi
    [[ -n "$DB_SCHEMA" ]] && args+=("--schema=$DB_SCHEMA")

    log_info "执行 DDL: $sql_file"
    if ! "$GENTOL_CMD" "${args[@]}"; then
        log_error "DDL 执行失败"
    fi
    log_info "DDL 执行完成!"
}

# 输出统一脚本入口的使用说明。
usage() {
    echo "用法:"
    echo "  $0                 生成 DAO/Model 代码"
    echo "  $0 ddl <sql文件>   执行 DDL 文件"
    exit 1
}

# 根据 ddl 子命令选择 DDL 或代码生成模式。
main() {
    if ! load_toml_config; then
        load_yaml_config || log_info "未找到数据库配置文件，仅使用外部环境变量"
    fi

    if [[ $# -eq 0 ]]; then
        check_gentol
        generate_code
    elif [[ "$1" == "ddl" && $# -eq 2 ]]; then
        check_gentol
        execute_ddl "$2"
    else
        usage
    fi
}

main "$@"
