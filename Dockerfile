# 仅面向 Linux 部署（linux/amd64、linux/arm64）；不提供 Windows 容器支持。
# 跨架构构建使用 buildx：docker buildx build --platform linux/arm64 -t <tag> .
#
# ============================
# Stage 1: 构建后端
# ============================
# 内网环境可改用私有仓库镜像：
#FROM iregistry.harbor.local/library/golang:1.25-bookworm AS backend-builder
FROM golang:1.25-bookworm AS backend-builder

USER work
WORKDIR /home/work
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# CGO_ENABLED=1 是因为 sqlite 驱动需要 cgo；构建镜像与运行镜像同为 debian 系，
# 保证 libc 一致（alpine 构建 + debian 运行会因 musl/glibc 不匹配而启动失败）。
# Go 版本与 go.mod、.golangci.yml 中的 1.25 保持一致。
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o bin/generate-example-project .

# ============================
# Stage 2: 运行镜像
# ============================
# 内网环境可改用私有仓库镜像：
#FROM iregistry.harbor.local/library/debian:bookworm-slim
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && rm -rf /var/lib/apt/lists/* \

USER work
WORKDIR /home/work

# 后端二进制与配置目录（服务启动时读取 ./conf/application.yaml）
COPY --from=backend-builder /home/work/bin/generate-example-project ./bin/
COPY --from=backend-builder /home/work/conf ./conf/

# 可选目录（如存在则拷入，需取消注释）
# COPY --from=backend-builder /home/work/data ./data/
# COPY --from=backend-builder /home/work/script ./script/
# COPY --from=backend-builder /home/work/docs ./docs/

# 配置注入：当前版本未实现环境变量覆盖，配置以 conf/application.yaml 为准。
# 需要按环境注入配置时，把 conf 目录挂为卷，或在启动前渲染该文件：
#   docker run -v ./conf:/home/work/conf generate-example-project:latest

# HTTP 服务端口（application.server.http.port，默认 8080）
EXPOSE 8080

ENTRYPOINT ["./bin/generate-example-project"]
