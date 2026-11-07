#!/usr/bin/env bash
#
# generate-example-project 一键启动脚本（Linux / Unix）
#
# 拉起后端服务；前端为可选能力：
#   1. 后端优先复用 bin/generate-example-project，缺失时自动 go build；
#   2. webroot/package.json 存在（接入了前端源码）时顺带启动 vite dev；
#   3. webroot 仅作为构建产物占位目录时，只启动后端。
#
# 用法:
#   ./start.sh          一键启动
#   ./start.sh -h       查看帮助
#
# 可用环境变量:
#   FRONTEND_PORT       前端端口，默认 9527，会作为 --port 传给 vite dev
#
# Ctrl+C 会同时关闭前后端进程。
# 注意：进程回收依赖 Unix 信号与进程组，Windows 请在 WSL / MSYS2 / Linux 环境使用。

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="${ROOT_DIR}/webroot"
BACKEND_BIN="${ROOT_DIR}/bin/generate-example-project"

# 端口以 conf/application.yaml 的 application.server.http.port 为准
BACKEND_PORT=8080
FRONTEND_PORT="${FRONTEND_PORT:-9527}"

# 记录所有子进程 pid，退出时统一回收
PIDS=()

usage() {
  # 只输出文件头部的连续注释块，遇第一个非注释行即停止；去掉开头的空行
  sed -n '2,${/^#/!q;s/^# \{0,1\}//;p}' "${BASH_SOURCE[0]}" | sed '/./,$!d'
}

log() {
  echo "[start] $*"
}

# spawn 在指定目录后台启动命令；优先用 setsid 建立独立进程组，便于整体回收
spawn() {
  local dir="$1"
  shift
  if command -v setsid >/dev/null 2>&1; then
    ( cd "${dir}" && exec setsid "$@" ) &
  else
    ( cd "${dir}" && exec "$@" ) &
  fi
  PIDS+=("$!")
}

# kill_tree 递归结束后代进程，避免多层派生的子进程残留
kill_tree() {
  local pid="$1"
  local sig="$2"
  local child
  [ -z "${pid}" ] && return 0
  if command -v pgrep >/dev/null 2>&1; then
    for child in $(pgrep -P "${pid}" 2>/dev/null || true); do
      kill_tree "${child}" "${sig}"
    done
  fi
  kill "-${sig}" "${pid}" >/dev/null 2>&1 || true
}

# stop_pid 优先按进程组结束（setsid 启动时进程组号等于 pid），否则退回递归结束
stop_pid() {
  local pid="$1"
  local sig="$2"
  kill "-${sig}" -- "-${pid}" >/dev/null 2>&1 && return 0
  kill_tree "${pid}" "${sig}"
}

# cleanup 先通知退出，超时未退则强杀，确保 Ctrl+C 不留孤儿进程
cleanup() {
  trap - EXIT INT TERM
  if [ "${#PIDS[@]}" -eq 0 ]; then
    return 0
  fi

  log "正在停止服务..."
  local pid
  for pid in "${PIDS[@]}"; do
    stop_pid "${pid}" TERM
  done

  # 给 2 秒优雅退出窗口，仍有存活进程则强杀
  sleep 2
  for pid in "${PIDS[@]}"; do
    stop_pid "${pid}" KILL
  done

  wait >/dev/null 2>&1 || true
}

# start_backend 编译（如有需要）并以后台进程方式启动后端
start_backend() {
  if [ ! -x "${BACKEND_BIN}" ]; then
    if ! command -v go >/dev/null 2>&1; then
      echo "[start] 未找到 ${BACKEND_BIN}，且当前环境没有 go 命令，无法启动后端" >&2
      exit 1
    fi
    log "未找到后端二进制，开始编译: ${BACKEND_BIN}"
    mkdir -p "$(dirname "${BACKEND_BIN}")"
    ( cd "${ROOT_DIR}" && go build -o "${BACKEND_BIN}" . )
  fi

  log "启动后端 -> http://localhost:${BACKEND_PORT}"
  # 后端依赖 conf/ 相对路径，必须在项目根目录启动
  spawn "${ROOT_DIR}" "${BACKEND_BIN}"
}

# start_frontend 仅在 webroot 内含前端工程时启动 vite dev server
start_frontend() {
  if [ ! -f "${WEB_DIR}/package.json" ]; then
    log "webroot 无前端工程（无 package.json），跳过前端，仅提供后端 API"
    return 0
  fi

  if ! command -v pnpm >/dev/null 2>&1; then
    echo "[start] 警告: 未找到 pnpm（需要 >=10.5），跳过前端启动，仅提供后端 API" >&2
    return 0
  fi

  if [ ! -d "${WEB_DIR}/node_modules" ]; then
    log "首次运行，安装前端依赖..."
    ( cd "${WEB_DIR}" && pnpm install )
  fi

  log "启动前端 -> http://localhost:${FRONTEND_PORT}"
  spawn "${WEB_DIR}" pnpm dev --port "${FRONTEND_PORT}"
}

main() {
  case "${1:-}" in
    -h | --help)
      usage
      exit 0
      ;;
    "") ;;
    *)
      echo "[start] 未知参数: $1（使用 -h 查看用法）" >&2
      exit 1
      ;;
  esac

  trap cleanup EXIT INT TERM

  start_backend
  start_frontend

  log "启动完成，Ctrl+C 结束全部服务"
  wait
}

main "$@"
