#!/bin/bash
# 模板改名脚本：将模板 module path 与服务名替换为新项目的值。
# 用法: bash script/rename.sh <new-service-name> <new-module-path>
# 示例: bash script/rename.sh my-service github.com/you/my-service

set -euo pipefail

OLD_MODULE="github.com/jasonlabz/generate-example-project"
OLD_NAME="generate-example-project"

NEW_NAME="${1:-}"
NEW_MODULE="${2:-}"

if [[ -z "$NEW_NAME" || -z "$NEW_MODULE" ]]; then
    echo "用法: bash script/rename.sh <new-service-name> <new-module-path>"
    echo "示例: bash script/rename.sh my-service github.com/you/my-service"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# 先替换完整 module path，再替换裸服务名（module 含服务名子串，顺序不能反）。
# 跳过 *.pb.go：生成代码内嵌带长度前缀的二进制描述符，纯文本替换会破坏
# 长度字节导致运行期 panic，改为在替换后重新生成刷新。
find . -type f \
    \( -name '*.go' -o -name '*.mod' -o -name '*.yaml' -o -name '*.yml' \
       -o -name '*.sh' -o -name '*.ps1' -o -name '*.md' -o -name '*.proto' \
       -o -name 'Makefile' -o -name 'Dockerfile' \) \
    -not -name '*.pb.go' \
    -not -path './.git/*' -not -path './bin/*' -not -path './output/*' -print0 |
    xargs -0 perl -pi -e "s{\Q$OLD_MODULE\E}{$NEW_MODULE}g; s{\Q$OLD_NAME\E}{$NEW_NAME}g"

# 重新生成 pb 代码，刷新生成文件内嵌描述符中的 go_package 元数据
# （不重新生成也能正常编译运行，仅描述符元数据保留旧 module 字符串）
if command -v protoc >/dev/null 2>&1; then
    bash "$SCRIPT_DIR/proto.sh" || echo "警告: pb 代码再生成失败，请稍后手动执行 bash script/proto.sh"
else
    echo "提示: 未检测到 protoc，跳过 pb 代码再生成，请安装后执行 bash script/proto.sh"
fi

echo "module 已替换为 ${NEW_MODULE}，服务名已替换为 ${NEW_NAME}"
echo "后续: go mod tidy && make build"
