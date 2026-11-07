# 仅面向 Linux 部署（linux/amd64、linux/arm64）；不提供 Windows 容器支持。
# 跨架构构建使用 buildx：docker buildx build --platform linux/arm64 -t <tag> .
#
# 结构：一个编译阶段 + 一个运行镜像，用 --target 选择：
#   docker build --target backend -t generate-example-project:latest .
#
# 编译阶段跑 make -f Makefile build frontend package，产物统一落在 output/，
# 运行镜像只从 output/ COPY，不自己编译。
#
# 若已在 Linux 宿主机上 make 过（产物在 output/），可加 --build-arg COMPILE=false
# 跳过编译，直接使用构建上下文里带过去的 output/：
#   make build frontend package && docker build --target backend --build-arg COMPILE=false .
# Windows 下 make 产出 .exe，不能用于 Linux 镜像，务必在 Linux/类 Unix 环境 make。
#
# ARG 必须写在文件最开头，夹在两个 FROM 之间不生效。
# 内网环境可把两个镜像 ARG 的默认值改成私有仓库等价镜像。
ARG BUILDER_IMAGE=docker.1ms.run/library/golang:1.25-bookworm
# 运行基础镜像：预装在 script/Dockerfile.runtime（通用发行版 + vi/tail/curl/ps/netstat/ping/dig/less），
# 一次构建长期复用，业务镜像不再每次 apt-get install。首次使用前执行 ./script/build-runtime.sh
ARG BACKEND_RUNTIME_IMAGE=runtime-go:bookworm
ARG COMPILE=true

# ============================
# Stage 1: 编译产物
# ============================
FROM ${BUILDER_IMAGE} AS build

WORKDIR /src
COPY . .

# 文件顶部的 ARG 在 stage 内不会自动生效，要在本 stage 重新声明一次
ARG COMPILE=true

# 只跑编译/打包，不跑 all：all 里含 test，容器内按需裁剪。
# Go 版本与 go.mod、.golangci.yml 中的 1.25 保持一致；CGO_ENABLED 沿用镜像默认(1)，
# 因 sqlite 驱动需要 cgo，且编译/运行镜像同为 debian 系可保证 libc 一致。
RUN if [ "${COMPILE}" = "true" ]; then \
        make -f Makefile build frontend package; \
    else \
        test -d output/bin || { \
            echo "ERROR: COMPILE=false needs output/ in build context, run 'make build frontend package' first" >&2; \
            exit 1; \
        }; \
    fi

# ============================
# Stage 2: 运行镜像
# ============================
FROM ${BACKEND_RUNTIME_IMAGE} AS backend

LABEL maintainer="generate-example-project"
LABEL description="generate-example-project 服务"

# 排查命令（vi/tail/curl/ps/netstat/ping/dig/less）与时区已在 runtime-go 基础镜像里
WORKDIR /home/work

COPY --from=build /src/output/bin/ ./bin/
COPY --from=build /src/output/conf/ ./conf/
# 可选目录（产物里存在时再拷）
# COPY --from=build /src/output/data/ ./data/
# COPY --from=build /src/output/docs/ ./docs/

# 配置注入：当前版本未实现环境变量覆盖，配置以 conf/application.yaml 为准。
# 需要按环境注入配置时，把 conf 目录挂为卷，或在启动前渲染该文件：
#   docker run -v ./conf:/home/work/conf generate-example-project:latest

# HTTP 服务端口（application.server.http.port，默认 8080）
EXPOSE 8080

ENTRYPOINT ["./bin/generate-example-project"]
