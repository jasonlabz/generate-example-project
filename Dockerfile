# ============================
# Stage 1: 构建后端
# ============================
FROM golang:1.26-alpine3.23 AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o bin/generate-example-project ./cmd/server

# ============================
# Stage 2: 运行镜像
# ============================
FROM debian:bullseye-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/bin/generate-example-project ./bin/
COPY --from=builder /app/conf ./conf/

# HTTP 服务端口
EXPOSE 8080
# gRPC 服务端口（application.server.grpc.enable=true 时生效）
EXPOSE 8082

ENTRYPOINT ["./bin/generate-example-project"]
