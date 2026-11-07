#!/bin/bash
# 在开发机（或任意装了 docker 的机器）上用编译镜像跑 make，产物落在 output/。
#
# 编译镜像只构建一次，这里只是 docker run 复用它；真正耗时的 go mod download
# 通过把缓存目录挂到宿主机来复用，第一次之后基本只有增量编译的时间。
#
# 用法:
#   ./script/dev-build.sh                  # make build frontend package
#   ./script/dev-build.sh build            # 只编译后端
#   ./script/dev-build.sh --shell          # 进容器手动排查
#
# 生成 output/ 后，可以直接用 --prebuilt 做镜像，不再重复编译：
#   ./docker-build.sh --prebuilt

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

BUILDER_IMAGE="${BUILDER_IMAGE:-dagine-builder:latest}"
CACHE_DIR="${DEV_BUILD_CACHE:-${HOME}/.cache/app-build}"
TARGETS=(build frontend package)
SHELL_MODE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --builder-image) BUILDER_IMAGE="${2:?缺少镜像名}"; shift 2 ;;
    --cache-dir)     CACHE_DIR="${2:?缺少缓存目录}"; shift 2 ;;
    --shell)         SHELL_MODE=true; shift ;;
    -h|--help)
      cat <<'EOF'
用法: ./script/dev-build.sh [选项] [make 目标...]

选项:
  --builder-image NAME  编译镜像（默认: dagine-builder:latest）
  --cache-dir DIR       依赖缓存目录（默认: ~/.cache/app-build）
  --shell               进容器交互 shell，手动排查
  -h, --help            显示帮助

不带目标时默认执行: make build frontend package
EOF
      exit 0
      ;;
    -*) echo "未知参数: $1" >&2; exit 1 ;;
    *)  if [ "${#TARGETS[@]}" -eq 3 ] && [ "${TARGETS[0]}" = "build" ] && [ "${TARGETS[1]}" = "frontend" ]; then
          TARGETS=("$@")
        else
          TARGETS+=("$1")
        fi
        shift ;;
  esac
done

if ! docker image inspect "${BUILDER_IMAGE}" >/dev/null 2>&1; then
  echo "错误: 编译镜像 ${BUILDER_IMAGE} 不存在" >&2
  echo "      可先构建: docker build -f script/Dockerfile.builder -t ${BUILDER_IMAGE} ." >&2
  exit 1
fi

mkdir -p "${CACHE_DIR}/gomod"

DOCKER_ARGS=(
  --rm
  -v "${ROOT_DIR}:/src"
  -w /src
  -v "${CACHE_DIR}/gomod:/root/go/pkg/mod"
)

if [ "${SHELL_MODE}" = true ]; then
  echo "进入容器: ${BUILDER_IMAGE}（源码挂载在 /src）"
  docker run -it "${DOCKER_ARGS[@]}" "${BUILDER_IMAGE}" sh
  exit 0
fi

echo "========================================="
echo " 编译镜像: ${BUILDER_IMAGE}"
echo " 依赖缓存: ${CACHE_DIR}/gomod"
echo " make 目标: ${TARGETS[*]}"
echo "========================================="

docker run "${DOCKER_ARGS[@]}" "${BUILDER_IMAGE}" make -f Makefile "${TARGETS[@]}"

echo ""
echo "产物已就绪: ${ROOT_DIR}/output"
echo "直接用它打镜像（不再编译）: ./docker-build.sh --prebuilt"
