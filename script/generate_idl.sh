#!/bin/bash

set -euo pipefail

# 配置变量
KITEX_CMD="${1:-kitex}"
BASE_MODULE="github.com/jasonlabz/generate-example-project"
ROOT_DIR=$(pwd)
IDL_DIR="idl"
CLIENT_DIR="client/kitex"
SERVER_DIR="server/kitex"

# 颜色输出函数
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 kitex 是否可用
check_kitex() {
    if ! command -v "$KITEX_CMD" &> /dev/null; then
        log_info "Installing kitex..."
        go install github.com/cloudwego/kitex/tool/cmd/kitex@latest
        if ! command -v "$KITEX_CMD" &> /dev/null; then
            log_error "kitex command not found and installation failed"
            exit 1
        fi
    fi
    log_info "Using kitex: $(which $KITEX_CMD)"
}

# 从文件路径提取服务名（不带后缀）
get_service_name() {
    local file_path="$1"
    local filename=$(basename "$file_path")
    local service_name="${filename%.*}"  # 移除文件后缀
    echo "$service_name"
}

# 通用生成函数
gen_kitex() {
    local type="$1"
    local msg="$2"
    local extra_args="$3"
    local idl_file="$4"
    local gen_path="${5:-}"
    local use_service="${6:-false}"  # 新增：是否使用服务名参数

    log_info "$msg"

    local base_args="-module $BASE_MODULE"
    if [[ -n "$gen_path" ]]; then
        base_args="$base_args -gen-path $gen_path"
    fi

    # 如果启用服务模式，从文件名生成服务名
    if [[ "$use_service" == "true" ]]; then
        local service_name=$(get_service_name "$idl_file")
        base_args="$base_args -service $service_name"
        log_info "Generating service: $service_name"
    fi

    if [[ ! -f "$idl_file" ]]; then
        log_warn "IDL file not found: $idl_file, skipping..."
        return 1
    fi

    log_info "Generating from: $idl_file"

    if ! $KITEX_CMD $base_args $extra_args "$idl_file"; then
        log_error "Failed to generate $type from $idl_file"
        return 1
    fi

    log_info "Successfully generated $type from $idl_file"
    return 0
}

# 遍历生成所有 thrift 文件（默认不加 -service）
gen_thrift() {
    log_info "Scanning for thrift files in $IDL_DIR/client..."

    if [[ ! -d "$IDL_DIR/client" ]]; then
        log_warn "IDL directory '$IDL_DIR/client' not found, skipping thrift generation"
        return 1
    fi

    if [[ ! -d "$CLIENT_DIR" ]]; then
        mkdir -p $CLIENT_DIR
    fi

    local thrift_files=()
    while IFS= read -r -d $'\0' file; do
        thrift_files+=("$file")
    done < <(find "$IDL_DIR/client" -name "*.thrift" -type f -print0)

    if [[ ${#thrift_files[@]} -eq 0 ]]; then
        log_warn "No thrift files found in $IDL_DIR/client"
        return 1
    fi

    log_info "Found ${#thrift_files[@]} thrift file(s):"
    for file in "${thrift_files[@]}"; do
        log_info "  - $file"
    done

    local success_count=0
    local fail_count=0

    for thrift_file in "${thrift_files[@]}"; do
        log_info "Processing: $thrift_file"

        if gen_kitex "thrift" "Generate thrift from $thrift_file..." "-thrift frugal_tag -invoker" "$thrift_file" $CLIENT_DIR; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    log_info "Thrift generation completed: $success_count successful, $fail_count failed"

    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# 生成 thrift 服务端代码（新增：带 -service 参数，基于文件名）
gen_thrift_service() {
    log_info "Scanning for thrift files in $IDL_DIR/server (with service mode)..."

    if [[ ! -d "$IDL_DIR/server" ]]; then
        log_warn "IDL directory '$IDL_DIR/server' not found, skipping thrift service generation"
        return 1
    fi

    if [[ ! -d "$SERVER_DIR" ]]; then
        mkdir -p $SERVER_DIR
    fi

    local thrift_files=()
    while IFS= read -r -d $'\0' file; do
        thrift_files+=("$file")
    done < <(find "$IDL_DIR/server" -name "*.thrift" -type f -print0)

    if [[ ${#thrift_files[@]} -eq 0 ]]; then
        log_warn "No thrift files found in $IDL_DIR/server for service generation"
        return 1
    fi

    local success_count=0
    local fail_count=0

    for thrift_file in "${thrift_files[@]}"; do
        log_info "Processing (service): $thrift_file"

        if gen_kitex "thrift service" "Generate thrift service from $thrift_file..." "-thrift frugal_tag -invoker" "$thrift_file" $SERVER_DIR "true"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    log_info "Thrift service generation completed: $success_count successful, $fail_count failed"

    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# 遍历生成所有 slim thrift 文件（默认不加 -service）
gen_thrift_slim() {
    log_info "Scanning for thrift files in $IDL_DIR/client (slim mode)..."

    if [[ ! -d "$IDL_DIR/client" ]]; then
        log_warn "IDL directory '$IDL_DIR/client' not found, skipping slim thrift generation"
        return 1
    fi

    if [[ ! -d "$CLIENT_DIR" ]]; then
        mkdir -p $CLIENT_DIR
    fi

    local thrift_files=()
    while IFS= read -r -d $'\0' file; do
        thrift_files+=("$file")
    done < <(find "$IDL_DIR/client" -name "*.thrift" -type f -print0)

    if [[ ${#thrift_files[@]} -eq 0 ]]; then
        log_warn "No thrift files found in $IDL_DIR/client for slim generation"
        return 1
    fi

    local success_count=0
    local fail_count=0

    for thrift_file in "${thrift_files[@]}"; do
        log_info "Processing (slim): $thrift_file"

        if gen_kitex "thrift(slim)" "Generate slim thrift from $thrift_file..." "-thrift frugal_tag -thrift template=slim" "$thrift_file" "$CLIENT_DIR/slim"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    log_info "Slim thrift generation completed: $success_count successful, $fail_count failed"

    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# 生成 slim thrift 服务端代码（新增：带 -service 参数，基于文件名）
gen_thrift_slim_service() {
    log_info "Scanning for thrift files in $IDL_DIR/server (slim mode with service)..."

    if [[ ! -d "$IDL_DIR/server" ]]; then
        log_warn "IDL directory '$IDL_DIR/server' not found, skipping slim thrift service generation"
        return 1
    fi

    if [[ ! -d "$SERVER_DIR" ]]; then
        mkdir -p $SERVER_DIR
    fi

    local thrift_files=()
    while IFS= read -r -d $'\0' file; do
        thrift_files+=("$file")
    done < <(find "$IDL_DIR/server" -name "*.thrift" -type f -print0)

    if [[ ${#thrift_files[@]} -eq 0 ]]; then
        log_warn "No thrift files found in $IDL_DIR/server for slim service generation"
        return 1
    fi

    local success_count=0
    local fail_count=0

    for thrift_file in "${thrift_files[@]}"; do
        log_info "Processing (slim service): $thrift_file"

        if gen_kitex "thrift(slim) service" "Generate slim thrift service from $thrift_file..." "-thrift frugal_tag -thrift template=slim" "$thrift_file" "$SERVER_DIR/slim" "true"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    log_info "Slim thrift service generation completed: $success_count successful, $fail_count failed"

    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Protobuf 生成函数（默认不加 -service）
gen_protobuf() {
    log_info "Scanning for proto files in $IDL_DIR/client..."

    if [[ ! -d "$IDL_DIR/client" ]]; then
        log_warn "IDL directory '$IDL_DIR/client' not found, skipping protobuf generation"
        return 1
    fi

    if [[ ! -d "$CLIENT_DIR" ]]; then
        mkdir -p $CLIENT_DIR
    fi

    local proto_files=()
    while IFS= read -r -d $'\0' file; do
        proto_files+=("$file")
    done < <(find "$IDL_DIR/client" -name "*.proto" -type f -print0)

    if [[ ${#proto_files[@]} -eq 0 ]]; then
        log_warn "No proto files found in $IDL_DIR/client"
        return 1
    fi

    log_info "Found ${#proto_files[@]} proto file(s):"
    for file in "${proto_files[@]}"; do
        log_info "  - $file"
    done

    local success_count=0
    local fail_count=0

    for proto_file in "${proto_files[@]}"; do
        log_info "Processing: $proto_file"

        if gen_kitex "protobuf" "Generate protobuf from $proto_file..." "" "$proto_file" $CLIENT_DIR; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    log_info "Protobuf generation completed: $success_count successful, $fail_count failed"

    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# 生成 protobuf 服务端代码（新增：带 -service 参数，基于文件名）
gen_protobuf_service() {
    log_info "Scanning for proto files in $IDL_DIR/server (with service mode)..."

    if [[ ! -d "$IDL_DIR/server" ]]; then
        log_warn "IDL directory '$IDL_DIR/server' not found, skipping protobuf service generation"
        return 1
    fi

    if [[ ! -d "$SERVER_DIR" ]]; then
        mkdir -p $SERVER_DIR
    fi

    local proto_files=()
    while IFS= read -r -d $'\0' file; do
        proto_files+=("$file")
    done < <(find "$IDL_DIR/server" -name "*.proto" -type f -print0)

    if [[ ${#proto_files[@]} -eq 0 ]]; then
        log_warn "No proto files found in $IDL_DIR/server for service generation"
        return 1
    fi

    local success_count=0
    local fail_count=0

    for proto_file in "${proto_files[@]}"; do
        log_info "Processing (service): $proto_file"

        if gen_kitex "protobuf service" "Generate protobuf service from $proto_file..." "" "$proto_file" $SERVER_DIR "true"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    log_info "Protobuf service generation completed: $success_count successful, $fail_count failed"

    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# 子模块重新生成函数
regenerate_submod() {
    local path="$1"
    local module="$2"
    local thrift_file="$3"
    local use_service="${4:-false}"  # 修改：是否使用服务模式

    if [[ ! -d "$path" ]]; then
        mkdir -p $path
    fi

    local original_dir=$(pwd)
    cd "$path"

    log_info "Executing kitex command in $path with module: $module and thrift file: $thrift_file"

    if [[ ! -f "$thrift_file" ]]; then
        log_warn "Thrift file not found: $thrift_file in $path, skipping..."
        cd "$original_dir"
        return 1
    fi

    # 构建 kitex 命令
    local kitex_cmd="$KITEX_CMD -module $module"
    if [[ "$use_service" == "true" ]]; then
        local service_name=$(get_service_name "$thrift_file")
        kitex_cmd="$kitex_cmd -service $service_name"
        log_info "Generating service: $service_name"
    fi
    kitex_cmd="$kitex_cmd $thrift_file"

    # 执行 kitex 命令
    if ! eval $kitex_cmd; then
        log_error "Failed to generate for $path"
        cd "$original_dir"
        return 1
    fi

    # 更新依赖
    log_info "Updating dependencies for $path..."
    go get github.com/cloudwego/kitex@latest
    go mod tidy

    cd "$original_dir"
    log_info "Successfully regenerated $path"
    return 0
}

# 主执行函数
main() {
    log_info "Starting IDL code generation..."
    check_kitex

    # 生成主 IDL 文件（默认不加 -service）
    gen_thrift
#    gen_thrift_slim
    gen_protobuf

    # 如果需要生成服务端代码，取消注释下面的调用：
     gen_thrift_service           # 生成 thrift 服务端（基于文件名）
     gen_thrift_slim_service      # 生成 slim thrift 服务端（基于文件名）
     gen_protobuf_service         # 生成 protobuf 服务端（基于文件名）

    # 子模块配置数组
    declare -a submodules=(
        # TODO: 模板格式-> $path:$module:$thrift_file:$use_service(可选)
        # 例如："basic/example_shop:example_shop:idl/item.thrift"           # 仅客户端
        # 例如："basic/example_shop:example_shop:idl/item.thrift:true"      # 服务端（基于文件名）
    )

    # 遍历所有子模块
    for submod in "${submodules[@]}"; do
        IFS=':' read -r path module thrift_file use_service <<< "$submod"
        regenerate_submod "$path" "$module" "$thrift_file" "$use_service"
    done

    log_info "All IDL code generation completed!"
}

# 执行主函数
main "$@"
