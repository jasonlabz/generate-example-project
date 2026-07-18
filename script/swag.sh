#!/bin/bash

set -euo pipefail

# 配置
SWAG_CMD="${SWAG_CMD:-swag}"
SWAG_VERSION="${SWAG_VERSION:-latest}"
AUTO_INSTALL_TOOLS="${AUTO_INSTALL_TOOLS:-false}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SWAG_DIR="${SWAG_DIR:-$PROJECT_ROOT/bin}"
PROJECT_DIR="${PROJECT_DIR:-$PROJECT_ROOT}"
SWAG_OUTPUT_DIR="${SWAG_OUTPUT_DIR:-docs/swagger}"
SWAG_MAIN_FILE="${SWAG_MAIN_FILE:-cmd/server/main.go}"
SWAG_PARSE_DEPTH="${SWAG_PARSE_DEPTH:-2}"
RUN_SWAG_FMT="${RUN_SWAG_FMT:-false}"

# 日志函数
log() { echo "[$(date '+%H:%M:%S')] $1"; }
log_info() { log "INFO: $1"; }
log_error() { log "ERROR: $1"; exit 1; }

# 检查依赖
check_swag() {
    if command -v "$SWAG_CMD" &>/dev/null; then
        return 0
    fi

    if [[ -f "$SWAG_DIR/swag" ]]; then
        SWAG_CMD="$SWAG_DIR/swag"
        return 0
    fi

    [[ "$AUTO_INSTALL_TOOLS" == "true" ]] || \
        log_error "swag 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true；SWAG_VERSION 默认 latest"
    [[ -n "$SWAG_VERSION" ]] || log_error "SWAG_VERSION 不能为空"

    log_info "Installing swag@$SWAG_VERSION..."
    go install "github.com/swaggo/swag/cmd/swag@$SWAG_VERSION"
    if ! command -v "$SWAG_CMD" &>/dev/null; then
        local go_bin
        go_bin="$(go env GOPATH)/bin/swag"
        [[ -x "$go_bin" ]] || log_error "swag 安装完成但当前 PATH 和 GOPATH/bin 中仍不可用"
        SWAG_CMD="$go_bin"
    fi
}

# 运行 swag 参数数组，避免通过字符串重新解析 shell 参数。
run_swag() {
    log_info "Running: swag $* (in $PROJECT_DIR)"

    (cd "$PROJECT_DIR" && "$SWAG_CMD" "$@") || log_error "swag $* failed"

    log_info "swag $* completed"
}

# 输出目录必须位于项目内，避免脚本覆盖外部路径。
validate_output_dir() {
    local normalized="${SWAG_OUTPUT_DIR//\\//}"
    [[ -n "$SWAG_OUTPUT_DIR" ]] || log_error "SWAG_OUTPUT_DIR 不能为空"
    [[ "$normalized" != /* && ! "$normalized" =~ ^[A-Za-z]:/ ]] || \
        log_error "SWAG_OUTPUT_DIR 必须是项目内相对路径"
    case "/$normalized/" in
        */../*|*/./../*) log_error "SWAG_OUTPUT_DIR 不得包含 .." ;;
    esac
}

usage() {
    echo "用法: $0 [--output <项目内相对目录>] [--main <入口文件>] [--format]"
    exit 1
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output) [[ $# -ge 2 ]] || usage; SWAG_OUTPUT_DIR="$2"; shift 2 ;;
            --main) [[ $# -ge 2 ]] || usage; SWAG_MAIN_FILE="$2"; shift 2 ;;
            --format) RUN_SWAG_FMT="true"; shift ;;
            -h|--help) usage ;;
            *) log_error "未知参数: $1" ;;
        esac
    done
}

# 主流程
main() {
    parse_args "$@"
    validate_output_dir
    log_info "Starting swag documentation generation..."

    check_swag || log_error "swag not found and installation failed"
    if [[ "$RUN_SWAG_FMT" == "true" ]]; then
        run_swag fmt
    fi
    run_swag init --generalInfo "$SWAG_MAIN_FILE" --output "$SWAG_OUTPUT_DIR" --parseDependency --parseDepth "$SWAG_PARSE_DEPTH"

    log_info "Documentation generation completed: $PROJECT_DIR/$SWAG_OUTPUT_DIR"
}

main "$@"
