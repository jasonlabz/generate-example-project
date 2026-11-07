#!/bin/bash
# Docker 镜像构建脚本（仅构建 Linux 镜像）
# 用法:
#   ./docker-build.sh              # 默认构建（镜像内跑 make -f Makefile build frontend package）
#   ./docker-build.sh -t v1.0.0    # 指定标签
#   ./docker-build.sh --no-cache   # 不使用缓存
#   ./docker-build.sh --prebuilt   # 不在镜像内编译，直接拷本机 make 产出的 output/
#   ./docker-build.sh --push       # 构建并推送

set -e

# ========== 配置 ==========
IMAGE_NAME="generate-example-project"
IMAGE_TAG=""
REGISTRY=""
PUSH=false
NO_CACHE=""
PLATFORM=""
# 非空时以 COMPILE=false 构建：不编译，直接拷 output/
COMPILE_ARG=""
# 运行基础镜像，留空用 Dockerfile 默认值
RUNTIME_IMAGE=""

# ========== 参数解析 ==========
while [[ $# -gt 0 ]]; do
  case $1 in
    -t|--tag)
      IMAGE_TAG="$2"; shift 2 ;;
    -r|--registry)
      REGISTRY="$2"; shift 2 ;;
    -p|--push)
      PUSH=true; shift ;;
    --no-cache)
      NO_CACHE="--no-cache"; shift ;;
    --platform)
      PLATFORM="--platform $2"; shift 2 ;;
    --backend-image)
      RUNTIME_IMAGE="$2"; shift 2 ;;
    --prebuilt)
      COMPILE_ARG="--build-arg COMPILE=false"; shift ;;
    -h|--help)
      echo "用法: $0 [选项]"
      echo ""
      echo "选项:"
      echo "  -t, --tag TAG        镜像标签 (默认: latest)"
      echo "  -r, --registry URL   Docker 镜像仓库地址"
      echo "  -p, --push           构建后推送到仓库"
      echo "  --no-cache           不使用构建缓存"
      echo "  --platform PLATFORM  多平台构建 (如 linux/amd64,linux/arm64)"
      echo "  --backend-image NAME 运行基础镜像 (默认: Dockerfile 内置 debian:bookworm)"
      echo "  --prebuilt           不在镜像内编译，直接拷本机 make 产出的 output/"
      echo "                       (需先执行 make build frontend package)"
      echo "  -h, --help           显示帮助"
      exit 0 ;;
    *)
      echo "未知参数: $1"; exit 1 ;;
  esac
done

# 默认标签
[ -z "$IMAGE_TAG" ] && IMAGE_TAG="latest"

# 拼接完整镜像名
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"
if [ -n "$REGISTRY" ]; then
  FULL_IMAGE="${REGISTRY}/${FULL_IMAGE}"
fi

echo "========================================="
echo " 构建镜像: ${FULL_IMAGE}"
echo "========================================="

# ========== 构建镜像 ==========
RUNTIME_BASE="${RUNTIME_IMAGE:-runtime-go:bookworm}"

# 运行基础镜像（含排查命令）由 script/build-runtime.sh 预构建，业务镜像只 FROM 它
if ! docker image inspect "${RUNTIME_BASE}" >/dev/null 2>&1; then
  echo "错误: 运行基础镜像 ${RUNTIME_BASE} 不存在" >&2
  echo "      先执行 ./script/build-runtime.sh 生成 runtime-go:bookworm" >&2
  exit 1
fi

# prebuilt 模式：产物必须已经在本机 make 出来，否则镜像会是空的
if [ -n "$COMPILE_ARG" ] && [ ! -d "output/bin" ]; then
  echo "错误: --prebuilt 需要 output/bin，请先执行 make build frontend package" >&2
  exit 1
fi

RUNTIME_ARG=""
if [ -n "$RUNTIME_IMAGE" ]; then
  RUNTIME_ARG="--build-arg BACKEND_RUNTIME_IMAGE=${RUNTIME_IMAGE}"
fi

BUILD_CMD="docker build ${NO_CACHE} ${PLATFORM} ${COMPILE_ARG} ${RUNTIME_ARG} -t ${FULL_IMAGE} -f Dockerfile --target backend ."

echo "执行: ${BUILD_CMD}"
echo ""
eval ${BUILD_CMD}

echo ""
echo "构建完成: ${FULL_IMAGE}"

# ========== 推送镜像 ==========
if [ "$PUSH" = true ]; then
  if [ -z "$REGISTRY" ]; then
    echo "错误: 推送镜像需要指定 --registry"
    exit 1
  fi
  echo "推送镜像: ${FULL_IMAGE}"
  docker push "${FULL_IMAGE}"
  echo "推送完成"
fi

echo ""
echo "运行示例:"
echo "  docker run -d -p 8080:8080 ${FULL_IMAGE}"
echo ""
echo "访问地址:"
echo "  Health:    http://localhost:8080/health-check"
echo "  API 文档:  http://localhost:8080/<服务名>/doc.html  (调试模式)"
