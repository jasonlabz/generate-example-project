#!/bin/bash
# 构建运行基础镜像（一次构建、长期复用）。
#
# 业务镜像只 FROM 它，不再每次 docker build 都跑 apt-get install，
# 省掉每次 30~60 秒的装包时间，也让业务镜像构建结果更稳定。
#
# 用法:
#   ./script/build-runtime.sh                          # 构建 runtime-go:bookworm
#   ./script/build-runtime.sh --registry harbor.local/base
#   ./script/build-runtime.sh --base debian:bookworm-slim
#
# 改了下面这些内容后重新执行即可，业务仓库不需要动：
#   新增排查命令、时区、基础发行版版本

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

REGISTRY=""
BASE_IMAGE="docker.1ms.run/library/debian:bookworm-slim"
# Debian 官方源在国内很慢，默认走镜像源；传 --apt-mirror "" 可用回官方源
APT_MIRROR="http://mirrors.aliyun.com"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -r|--registry) REGISTRY="${2:?缺少仓库地址}"; shift 2 ;;
    --base)        BASE_IMAGE="${2:?缺少基础镜像}"; shift 2 ;;
    --apt-mirror)  APT_MIRROR="${2-}"; shift 2 ;;
    -h|--help)
      cat <<'EOF'
用法: ./script/build-runtime.sh [选项]

选项:
  -r, --registry URL   镜像仓库前缀（如 harbor.local/base），留空则只打本地标签
  --base IMAGE         运行基础镜像的底座（默认: docker.1ms.run/library/debian:bookworm-slim）
  --apt-mirror URL     apt 镜像源前缀，留空串则用 Debian 官方源
                       （默认: http://mirrors.aliyun.com）
  -h, --help           显示帮助
EOF
      exit 0
      ;;
    *) echo "未知参数: $1" >&2; exit 1 ;;
  esac
done

if [ -n "${REGISTRY}" ]; then
  GO_IMAGE="${REGISTRY}/runtime-go:bookworm"
else
  GO_IMAGE="runtime-go:bookworm"
fi

echo "========================================="
echo " 构建: ${GO_IMAGE}  (base: ${BASE_IMAGE})"
echo "========================================="
docker build \
  -f script/Dockerfile.runtime \
  --build-arg "BASE_IMAGE=${BASE_IMAGE}" \
  --build-arg "APT_MIRROR=${APT_MIRROR}" \
  -t "${GO_IMAGE}" \
  .

echo ""
echo "完成: ${GO_IMAGE}"
echo "业务镜像直接 FROM 它即可："
echo "  docker build --target backend -t generate-example-project:latest ."
