#!/bin/bash

set -euo pipefail

AUTO_INSTALL_TOOLS="${AUTO_INSTALL_TOOLS:-false}"
PROTOC_GEN_GO_VERSION="${PROTOC_GEN_GO_VERSION:-latest}"
PROTOC_GEN_GO_GRPC_VERSION="${PROTOC_GEN_GO_GRPC_VERSION:-latest}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

log() { echo "[$(date '+%H:%M:%S')] $1"; }
log_info() { log "INFO: $1"; }
log_error() { log "ERROR: $1"; exit 1; }

check_protoc() {
    command -v protoc &>/dev/null || \
        log_error "protoc 不存在。macOS: brew install protobuf；Debian/Ubuntu: apt install -y protobuf-compiler；或从 github.com/protocolbuffers/protobuf/releases 下载"
}

check_plugin() {
    local cmd="$1" pkg="$2" ver="$3"
    if command -v "$cmd" &>/dev/null; then
        return
    fi
    local go_bin
    go_bin="$(go env GOPATH)/bin/$cmd"
    if [[ -x "$go_bin" ]]; then
        return
    fi
    [[ "$AUTO_INSTALL_TOOLS" == "true" ]] || \
        log_error "$cmd 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true"
    log_info "Installing $pkg@$ver..."
    go install "$pkg@$ver"
    [[ -x "$go_bin" ]] || command -v "$cmd" &>/dev/null || \
        log_error "$cmd 安装完成但当前 PATH 和 GOPATH/bin 中仍不可用"
}

main() {
    check_protoc
    check_plugin protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go "$PROTOC_GEN_GO_VERSION"
    check_plugin protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc "$PROTOC_GEN_GO_GRPC_VERSION"

    export PATH="$PATH:$(go env GOPATH)/bin"

    cd "$PROJECT_ROOT"
    local protos
    protos=$(find api/proto -name '*.proto')
    [[ -n "$protos" ]] || log_error "api/proto 下没有 .proto 文件"

    log_info "Generating protobuf code..."
    # shellcheck disable=SC2086
    protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        $protos
    log_info "Generation completed"
}

main "$@"
