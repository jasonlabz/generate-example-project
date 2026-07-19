# generate-example-project

基于 Gin 的标准项目模板：社区标准布局（`cmd/` + `internal/` + `pkg/`）、多入口、gRPC 示例、DAO 生成、完整工程化配套。

## 快速开始

```shell
## 1. 复制模板并改名（替换 module path 与服务名）
bash script/rename.sh my-service github.com/you/my-service

## 2. 修改配置
vi conf/application.yaml     # 服务端口、数据库、Redis 等，默认只开 HTTP :8080

## 3. 构建与运行
make build && ./bin/my-service
curl http://127.0.0.1:8080/health-check
```

## 目录结构

```
├── cmd/                  # 二进制入口：server（主服务）/ migrate / worker / tools
├── internal/             # 私有代码（编译器保证外部无法 import）
│   ├── bootstrap/        # 配置加载与资源初始化编排
│   ├── server/           # HTTP/gRPC/pprof 启动与优雅退出
│   ├── router/           # 路由注册
│   ├── controller/       # HTTP 入参适配层
│   ├── rpc/              # gRPC 服务实现
│   ├── service/          # 业务层（接口 + GetService() 单例实现 + dto）
│   ├── dal/db/           # 数据访问层（gentol 生成 dao/model 落点）
│   ├── client/           # 外部服务调用（配合 conf/servicer）
│   ├── middleware/       # gin 中间件
│   └── resource/         # 全局客户端单例
├── pkg/                  # 可复用库：ginx（响应封装）/ consts / helper
├── api/proto/            # proto 定义与生成物
├── conf/                 # application.yaml、migrations、seed、servicer
├── docs/swagger/         # swag 生成产物
└── script/               # gentol / swag / proto / rename 脚本
```

## 分层规则（强制）

1. controller/rpc 只做入参校验和响应组装，不写业务逻辑、不碰 dao
2. service 不 import gin
3. dao 不 import service
4. pkg 不 import internal

新增模块套路（以 `user` 为例）：

```
internal/service/user.go            # 接口
internal/service/user/impl.go       # 实现（GetService() 单例）
internal/service/user/dto.go        # 请求/响应 DTO
internal/controller/user.go         # HTTP 适配
internal/router/router.go           # 注册路由
```

## 常用命令

| 命令 | 说明 |
|------|------|
| `make build` | 编译主服务到 `bin/` |
| `make build-all` | 编译全部入口（server/migrate/worker） |
| `make test` | 单元测试（-race + 覆盖率） |
| `make lint` | golangci-lint 检查 |
| `make package` | 打包 bin + conf 到 `output/` |
| `make docker` | 构建 Docker 镜像 |

## 工具脚本

### gentol：DAO/Model 生成与 DDL 执行

若 `conf/db/<DB_CONF>`（默认 `db.toml`）存在则读取该文件，否则读 `conf/application.yaml`；环境变量始终优先。生成物落在 `internal/dal/db/{dao,model}`。

```shell
go install github.com/jasonlabz/gentol@master

export DB_TYPE=postgres DB_HOST=127.0.0.1 DB_PORT=5432 \
       DB_USER=postgres DB_PASS='your-password' DB_NAME=example DB_SCHEMA=public

bash script/gentol.sh                                          # 生成 DAO/Model
bash script/gentol.sh ddl conf/migrations/20240701_001_example_add_column.sql   # 执行 DDL
```

PowerShell 使用 `./script/gentol.ps1`。完整参数见 [script/README.md](script/README.md)。

### swag：接口文档生成

```shell
go install github.com/swaggo/swag/cmd/swag@v1.8.12
bash script/swag.sh        # 入口 cmd/server/main.go，输出 docs/swagger
```

debug 模式下访问 `http://ip:port/<服务名>/doc.html`（knife4go 美化版 swagger-ui）。

### proto：gRPC 代码生成

```shell
brew install protobuf      # 或 apt install protobuf-compiler
AUTO_INSTALL_TOOLS=true bash script/proto.sh
```

生成物与 `.proto` 同目录（`api/proto/<svc>/v1/`），已提交入库，不装 protoc 也能编译项目。

## 服务启用规则

`application.server.http.enable` 默认 `true`；`application.server.grpc.enable` 默认 `false`，需显式配置为 `true` 才会启动；pprof 由 `application.monitor.pprof` 控制，独立端口。

## 测试写法示例

| 层 | 示例文件 | 方式 |
|----|---------|------|
| dao | `internal/dal/db/dao/user_test.go` | `NewUserDao(内存 SQLite)` |
| service | `internal/service/health_check/impl_test.go` | 直接调用 |
| controller | `internal/controller/health_check_test.go` | httptest |
| rpc | `internal/rpc/hello_test.go` | bufconn |
