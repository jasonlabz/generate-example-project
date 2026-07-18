# generate-example-project 目录结构标准化设计

日期：2026-07-19
状态：已评审通过，待实施

## 1. 背景与目标

generate-example-project 是 Gin 项目模板，新项目（如 dagine-dashboard）由它复制改名而来。当前模板存在以下问题：

1. 根 `main.go` 与 `cmd/example-server/main.go` 几乎完全重复（各 ~237 行），启动逻辑已发生漂移（shutdown 超时一个 3s 一个 5s）
2. 缺少 `dal/` 数据访问层骨架，gentol 生成物没有预置落点，bootstrap 中 `dao.SetGormDB(db)` 处于注释状态
3. `server/README.md` 约定的 `body/request.go + response.go` 与实际代码 `dto.go` 不一致
4. 有 `conf/servicer/` 配置和 `initServicer` 机制，但没有对应的 client 调用示例
5. 零测试、无 lint 配置、无 CI、模板改名靠手工全局替换

目标：将模板重构为一套标准规范的初始化项目，向 Go 社区标准布局靠拢，同时保留现有 potato 生态的使用习惯。

## 2. 决策记录

| 决策点 | 结论 | 理由 |
|--------|------|------|
| 布局方向 | 社区标准布局（`internal/` + `pkg/` + `cmd/`） | `internal/` 由编译器保证私有代码不被外部 import；`pkg/` 明确可复用边界 |
| 依赖组织 | service 层保留全局单例 + `GetService()`；dao 层改构造装载模式 | 与现有生态（dagine-dashboard、potato）习惯一致，同时让 dao 层可注入测试库做单测 |
| 功能范围 | 保留 worker/migrate/tools 多入口、gRPC（补 proto 示例）；移除 web/ 前端骨架、独立静态文件服务器 | 模板定位纯后端，精简不常用能力 |
| 工程化配套 | 改名脚本、golangci-lint、示例单测、GitHub Actions CI、`.claude/rules/backend.md` 全部纳入 | 新项目从第一天具备完整工程化基线 |
| gRPC 实现包名 | `internal/rpc/` | 包名 `grpc` 与 `google.golang.org/grpc` 冲突，避免到处写别名 |
| dal 层级 | `internal/dal/db/{dao,model}` 保留 `db` 一层 | 与 dagine-dashboard 一致，未来可扩展 `dal/redis/`、`dal/es/` |
| proto 生成物 | 与 `.proto` 同目录（`api/proto/<svc>/v1/`） | import path 即目录路径，无需单独 `gen/` 目录 |
| DTO 命名 | 统一 `dto.go` | 以代码现状为准，废弃文档中 `body/` 旧约定 |

## 3. 目录结构

```
generate-example-project/
├── cmd/                              # 所有二进制入口（根目录 main.go 删除）
│   ├── server/main.go                # 主服务入口（瘦，<40 行，含 swagger 注释）
│   ├── migrate/main.go               # 手动执行数据库迁移
│   ├── worker/main.go                # 后台任务进程
│   └── tools/
│       ├── backfill/main.go          # 一次性回填脚本
│       └── fix/main.go               # 一次性修复脚本
├── internal/                         # 私有代码，编译器保证外部无法 import
│   ├── bootstrap/                    # 初始化编排
│   │   ├── bootstrap.go              # MustInit：config→logger→crypto→db→dao 装载→redis→...
│   │   ├── config.go
│   │   └── migrate.go                # ensureDB / runMigrations / runSeed
│   ├── resource/                     # 全局客户端单例（logger/RMQ/Redis/ES）
│   ├── server/                       # 服务器启动与优雅退出
│   │   ├── server.go                 # Run(ctx)：编排各 server + 统一 shutdown
│   │   ├── http.go                   # gin engine 装配 + HTTP server
│   │   ├── grpc.go                   # gRPC server（keepalive、调 rpc.Register）
│   │   └── pprof.go                  # pprof 独立端口
│   ├── router/                       # 路由注册
│   │   └── router.go
│   ├── controller/                   # HTTP 入参适配层
│   │   └── health_check.go
│   ├── rpc/                          # gRPC 服务实现（proto 接口实现）
│   │   └── hello.go
│   ├── service/                      # 业务层（接口 + GetService() 单例）
│   │   ├── health_check.go           # 接口定义
│   │   └── health_check/
│   │       ├── impl.go
│   │       └── dto.go
│   ├── dal/                          # 数据访问层（gentol 落点）
│   │   └── db/
│   │       ├── dao/                  # NewXxxDao(db) 构造 + SetGormDB 装载
│   │       └── model/
│   ├── client/                       # 外部服务调用（配合 conf/servicer）
│   │   └── demo/
│   └── middleware/                   # 项目自有中间件
├── api/
│   └── proto/hello/v1/hello.proto    # proto 定义，生成的 .pb.go 放同目录
├── pkg/                              # 可复用库，不依赖 internal
│   ├── ginx/                         # response / page
│   ├── consts/
│   └── helper/
├── conf/                             # application.yaml / log / migrations / seed / servicer
├── docs/swagger/                     # swag 生成产物
├── script/
│   ├── gentol.sh|.ps1                # 默认落点改为 internal/dal/db/*
│   ├── swag.sh|.ps1                  # -g 指向 cmd/server/main.go
│   ├── proto.sh|.ps1                 # protoc 生成脚本
│   └── rename.sh|.ps1                # 模板改名脚本
├── .claude/rules/backend.md          # 后端分层规范（AI 协作规则）
├── .github/workflows/ci.yml          # build + test + lint
├── .golangci.yml
├── Makefile                          # build 指向 ./cmd/server，新增 build-all、lint
├── Dockerfile                        # 构建 cmd/server
└── README.md                         # 目录说明 + 快速开始 + 分层规则 + 工具使用
```

## 4. 分层规则与依赖方向

依赖方向（单向，禁止反向）：

```
cmd/server ──→ internal/bootstrap ──→ internal/server ──→ internal/router
                                                              │
                              ┌───────────────┬───────────────┤
                              ▼               ▼               ▼
                        middleware      controller         (rpc 由 server/grpc.go 注册)
                                              │               │
                                              ▼               ▼
                                        internal/service ←────┘
                                              │
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                        dal/db/dao      internal/client   resource
                              │
                              ▼
                          gorm(自持 db)

pkg/* ← 任何层都可用；pkg 自身不 import internal
```

四条硬规则（写入 README 与 .claude/rules/backend.md）：

1. controller/rpc 只做入参校验和响应组装，不写业务逻辑、不碰 dao
2. service 不 import gin（入参用 dto，保持可脱离 HTTP 测试）
3. dao 不 import service（防循环依赖）
4. pkg 不 import internal（保证 pkg 可被抽走复用）

## 5. dao 装载模式与 service 单例

### dao 装载模式

```go
// internal/dal/db/dao/base.go
var defaultDB *gorm.DB

func SetGormDB(db *gorm.DB) { defaultDB = db }   // bootstrap 初始化 DB 后调用一次

// internal/dal/db/dao/user.go（gentol 生成或手写，模式一致）
type UserDao struct{ db *gorm.DB }

func NewUserDao(db *gorm.DB) *UserDao { return &UserDao{db: db} }  // 单测入口：传内存 SQLite
func GetUserDao() *UserDao            { return NewUserDao(defaultDB) }  // service 里使用
```

- 业务代码：`dao.GetUserDao().SelectByID(ctx, id)`
- 单测代码：`dao.NewUserDao(sqliteDB)`，不碰真库
- bootstrap 中 `dao.SetGormDB(db)` 从注释变为真实调用

### service 单例

保留 `GetService()` + `sync.Once`，修掉现有实现的竞态（`if svc != nil` 提前返回绕过 `once.Do`，并发首调可能拿到未初始化完成的实例）：

```go
var (
    svc  *Service
    once sync.Once
)

func GetService() service.HealthCheckService {
    once.Do(func() { svc = &Service{} })   // 不写 if svc != nil 提前返回
    return svc
}
```

## 6. 入口与启动

### cmd/server/main.go（~40 行）

```go
// @title    xxx服务
// @version  1.0
// ...swagger 注释...
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    bootstrap.MustInit(ctx)
    server.Run(ctx)          // 阻塞直到收到退出信号，内部完成优雅退出
}
```

### internal/server 职责拆分

| 文件 | 职责 |
|------|------|
| `server.go` | `Run(ctx)`：按配置起各 server → 等信号 → 统一 5s 超时优雅退出 |
| `http.go` | gin engine 装配（mode、metrics 中间件）+ HTTP server |
| `grpc.go` | gRPC server（keepalive 参数）+ 调 `rpc.Register(srv)` 注册所有 proto 实现 |
| `pprof.go` | pprof 独立端口 |

三个顺带修复：

1. 启动逻辑只保留一份，shutdown 超时统一 5s（现状两份 main 分别为 3s/5s）
2. pprof 只保留独立端口方式（现状既注册 gin 路由又起独立端口，二者重复）
3. 移除 `staticRouter.Static("/", "webroot")` 静态路由（随 web/ 一起删除）

worker/migrate/tools 入口保持现有瘦结构，仅更新 import 路径。

## 7. 工程化配套

1. **script/rename.sh|.ps1**：`bash script/rename.sh my-service github.com/you/my-service`。替换范围：go.mod module、全部 .go import、Makefile TARGETNAME、Dockerfile、docker-build.sh、conf/application.yaml 服务名、README 标题
2. **.golangci.yml + Makefile lint 目标**：基线启用 govet / staticcheck / errcheck / revive / gofmt / goimports
3. **示例单测**（三层各一，同时解决 dal 空目录问题）：

   | 测试 | 方式 | 演示什么 |
   |------|------|---------|
   | `internal/dal/db/dao/user_test.go` | 内存 SQLite + `NewUserDao(db)` | dao 装载模式的单测写法（模板自带手写 User model+dao 示例，风格对齐 gentol 产物） |
   | `internal/service/health_check/impl_test.go` | 纯函数调用 | service 单测写法 |
   | `internal/controller/health_check_test.go` | httptest + router | HTTP 层集成测试写法 |

4. **.github/workflows/ci.yml**：push/PR 触发 build + test + golangci-lint
5. **Makefile / Dockerfile**：build 指向 `./cmd/server`；新增 build-all（编译全部 cmd）、lint；删除 frontend / copy-frontend / webroot 目标；Dockerfile 同步
6. **README 重写**：目录结构职责表、快速开始（rename → 改配置 → 起服务）、四条分层硬规则、工具使用（gentol/swag/proto）
7. **.claude/rules/backend.md**：内容为第 4 节分层规则，模板新建的项目天生带 AI 协作规范

## 8. 迁移清单

### 平移（改路径 + 改 import）

| 现位置 | 新位置 |
|--------|--------|
| `bootstrap/` | `internal/bootstrap/` |
| `global/resource/` | `internal/resource/` |
| `server/routers/` | `internal/router/` |
| `server/controller/` | `internal/controller/` |
| `server/service/` | `internal/service/` |
| `server/middleware/` | `internal/middleware/` |
| `common/ginx/` | `pkg/ginx/` |
| `common/consts/` | `pkg/consts/` |
| `common/helper/` | `pkg/helper/` |
| `cmd/migrate、worker、tools` | 原地不动，仅更新 import |

### 删除

| 目标 | 原因 |
|------|------|
| 根 `main.go` | 启动逻辑收编进 internal/server，入口归 cmd/server |
| `cmd/example-server/` | 与根 main.go 重复 |
| `web/`、webroot 静态路由、startFileServer + basicAuth | 功能范围决策移除 |
| `server/README.md` | 内容修正后并入根 README（body/ 约定废弃，以 dto.go 为准） |
| Makefile frontend / copy-frontend 目标 | 无前端 |

### 新建

| 文件 | 内容 |
|------|------|
| `cmd/server/main.go` | 瘦入口 + swagger 注释 |
| `internal/server/{server,http,grpc,pprof}.go` | 启动与优雅退出 |
| `internal/dal/db/dao/base.go` + `user.go` + `user_test.go` | 装载模式基座 + 示例 dao + 单测 |
| `internal/dal/db/model/user.go` | 示例 model |
| `internal/rpc/hello.go` | proto 服务实现示例 |
| `internal/client/demo/demo.go` | servicer 调用示例（httpx.GetClient） |
| `api/proto/hello/v1/hello.proto` | proto 示例 |
| `script/proto.sh|.ps1`、`script/rename.sh|.ps1` | 生成/改名脚本 |
| `.golangci.yml`、`.github/workflows/ci.yml` | lint + CI |
| `.claude/rules/backend.md` | 分层规范 AI 规则 |
| `internal/service/health_check/impl_test.go`、`internal/controller/health_check_test.go` | 示例单测 |

### 联动修改

- `script/gentol.sh|.ps1`：MODEL_DIR/DAO_DIR 默认值改 `internal/dal/db/*`
- `script/swag.sh|.ps1`：`-g` 指向 `cmd/server/main.go`
- Makefile：build 指向 `./cmd/server`，新增 build-all、lint
- Dockerfile：构建路径同步
- `internal/bootstrap/bootstrap.go`：`dao.SetGormDB(db)` 取消注释成真实调用；`initServicer` 保持不变

## 9. 验收标准

1. `make build`、`make test`（三个示例测试通过）、`make lint` 全绿
2. `bash script/rename.sh demo github.com/x/demo` 后再次 `make build` 依然通过
3. gRPC enable 后 hello 示例服务可调通；HTTP `/health-check` 正常响应
