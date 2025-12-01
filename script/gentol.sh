#!/bin/bash

set -euo pipefail

# 配置参数
GENTOL_CMD="${GENTOL_CMD:-gentol}"
OUTPUT_DIR="${OUTPUT_DIR:-.}"
TEMPLATE_DIR="${TEMPLATE_DIR:-./template}"
DSN=""
# 数据库配置 #TODO: 修改对应参数
# TODO: 数据库类型  "mysql|postgres|sqlserver|oracle|sqlite|dm"
DB_TYPE="postgres"
# TODO: 数据库host
DB_HOST="****************"
# TODO: 数据库port
DB_PORT="8530"
# TODO: 数据库 用户
DB_USER="postgres"
# TODO: 数据库 密码
DB_PASS="****************"
# TODO: 数据库 库名
DB_NAME="database"
# TODO: 数据库 模式
DB_SCHEMA=""
# TODO: 需要生成的表结构，不配置则为全部
TABLES="${TABLES:-}"

# 生成配置
MODEL_DIR="${MODEL_DIR:-dal/db/model}"
DAO_DIR="${DAO_DIR:-dal/db/dao}"


# 功能开关
ONLY_MODEL="${ONLY_MODEL:-false}"
USE_SQL_NULLABLE="${USE_SQL_NULLABLE:-false}"
RUN_GOFMT="${RUN_GOFMT:-true}"
GEN_HOOK="${GEN_HOOK:-true}"

# 日志函数
log() { echo "[$(date '+%H:%M:%S')] $1"; }
log_info() { log "INFO: $1"; }
log_error() { log "ERROR: $1"; exit 1; }

# 构建 DSN（如果未直接提供）
build_dsn() {
    if [[ -n "$DSN" ]]; then
        return 0
    fi
    if [[ "$DSN" != "" ]]; then
        return 0
    fi

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
        *)
            log_error "Unsupported database type: $DB_TYPE"
            ;;
    esac
}

# 构建命令参数
build_args() {
    local args=(
        "--db_type=$DB_TYPE"
        "--dsn=\"$DSN\""
        "--model=$MODEL_DIR"
        "--dao=$DAO_DIR"
    )

    [[ -n "$TABLES" ]] && args+=("--table=$TABLES")
    [[ -n "$DB_SCHEMA" ]] && args+=("--schema=$DB_SCHEMA")
    [[ "$ONLY_MODEL" == "true" ]] && args+=("--only_model")
    [[ "$USE_SQL_NULLABLE" == "true" ]] && args+=("--use_sql_nullable")
    [[ "$RUN_GOFMT" == "true" ]] && args+=("--rungofmt")
    [[ "$GEN_HOOK" == "true" ]] && args+=("--gen_hook")

    echo "${args[@]}"
}

# 检查依赖
check_gentol() {
    if command -v "$GENTOL_CMD" &>/dev/null; then
        return 0
    fi

    log_info "Installing gentol..."
    go install github.com/jasonlabz/gentol@master
}

# 主流程
main() {
    log_info "Starting code generation with gentol..."

    check_gentol
    build_dsn

    local args
    args=$(build_args)

    log_info "Running: $GENTOL_CMD $args"

    if ! eval $GENTOL_CMD $args; then
        log_error "Code generation failed"
    fi

    log_info "Code generation completed!"
}

main "$@"
