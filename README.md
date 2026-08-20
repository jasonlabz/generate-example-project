# generate-example-project

基于 **gin + Huma v2** 的分层 API 示例项目。本项目同时是「代码模板」：
给开发者与大模型写代码时参考，覆盖路由、分层、文档、错误处理、测试与工具链的完整约定。

> 服务启用规则：`application.server.http.enable` 默认 `true`；`application.server.grpc.enable`、`application.server.static.enable` 默认 `false`，需显式配置为 `true` 才会启动。

## 快速开始

```shell
go run main.go
```

- Knife4j 文档 UI（调试模式）：`http://127.0.0.1:<port>/<服务名>/doc.html`
- OpenAPI 3.0 文档：`http://127.0.0.1:<port>/<服务名>/v3/api-docs`
- 示例接口：`GET http://127.0.0.1:<port>/<服务名>/health-check`

## 目录结构（分层架构）

```text
main.go                    启动入口：HTTP/GRPC/pprof/Prometheus、优雅退出
bootstrap/                 初始化：配置、日志、DB、迁移、种子数据、RMQ、Redis
cmd/                       子命令入口（example-server / migrate / tools / worker）
common/                    跨层通用能力
├── humax/                 huma 响应、分页、文件流与错误封装
├── ginx/                  gin Context 写入适配（复用 humax 响应类型）
└── consts/ helper/        常量与辅助函数
server/
├── router/router.go       路由组装：huma 组层级 + 中间件 + knife4go 文档
├── controller/            控制器层：huma 路由注册 + DTO 定义 + 转换
├── service/               服务层：业务逻辑入口（用例编排）
├── manager/               技术能力层：外部系统/基础设施访问（DB、Redis 等）
├── middleware/            gin 中间件（上下文、日志）
└── wire/                  依赖组装（NewController/NewService/NewManager）
mocks/                     mockgen 生成的接口 mock
conf/                      配置（application.yaml、日志、迁移 SQL）
script/                    gentol（DAO/Model 生成）、go-mockgen 脚本
docs/                      设计文档、开发过程记录
```

### 分层职责（强制）

| 层 | 职责 | 禁止 |
|----|------|------|
| controller | 路由注册、参数/响应模型、DTO 与业务模型转换 | 业务逻辑、直连 DB |
| service | 业务用例编排（调用 manager 组合能力） | 直接访问 DAO/外部系统 |
| manager | 单一技术能力（DB 访问、Redis、外部 API） | 业务判断 |
| wire | 依赖装配 | 逻辑 |

调用链：`HTTP → controller → service → manager → DAO/外部系统`

## Huma 编写规范（新增业务模块模板）

新增模块（如 `user`）按以下步骤，文件结构与 `server/controller/health_check/` 保持一致：

### 1. 控制器：`server/controller/<module>/register.go`

```go
// Register 把模块的 HTTP 操作注册到 api（huma 组）。
// huma.Register 三要素：
//  1. api：huma.API 或 huma.NewGroup（组自动应用前缀）；
//  2. huma.Operation：OpenAPI 元信息（Summary/Tags/OperationID）；
//  3. handler：func(ctx context.Context, in *In) (*Out, error)。
func (c *controllerImpl) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "user-get",
		Method:      http.MethodGet,
		Path:        "/users/{id}",        // 路径参数用 {name} 占位
		Summary:     "获取用户",
		Tags:        []string{"用户"},
	}, c.handleGet)
}
```

### 2. 请求/响应模型：`types.go`

```go
// 请求参数结构体用 huma tag 声明位置与校验，huma 自动生成 OpenAPI 参数。
type getUserInput struct {
	ID int64 `path:"id" example:"1"`     // path/query/header/body 四类位置
}

type userOutput struct {
	Body *humax.Envelope[*userVO]        // 响应体统一用 humax.Envelope
}
```

### 3. 处理器：`<module>_controller.go`

```go
// handleGet 是 huma 处理器：入参结构体、出参结构体、error（状态码）。
func (c *controllerImpl) handleGet(ctx context.Context, in *getUserInput) (*userOutput, error) {
	user, err := c.service.GetUser(ctx, in.ID)
	if err != nil {
		return nil, humax.NotFoundError(consts.APIVersionV1, err)  // 带状态码的错误
	}
	return &userOutput{Body: humax.New(consts.APIVersionV1, toUserVO(user))}, nil
}
```

### 4. 业务层

命名与边界（service/manager/controller 三层一致，以 health_check 为示范）：

| 项 | 规则 | 示例 |
|----|------|------|
| 包边界 | `<module>` 目录就是业务域包；不按接口或路由数量拆包 | `health_check` |
| Service | 一个接口承载同一业务规则、依赖图和生命周期内的多个用例方法 | `Service.Check` / `Service.CheckReadiness` |
| Manager | 一个接口承载共享技术依赖的多个技术操作 | `Manager.Check` / `Manager.CheckReadiness` |
| Controller | 默认使用具体 `Controller`，不为路由注册额外抽象接口 | `NewController(service.Service) *Controller` |
| 构造函数 | 域名由包路径表达，默认用简洁的 `NewService`、`NewManager`、`NewController` | `service.NewService(...)` |
| 多协作者 | 只有出现独立依赖图、事务边界或生命周期时，才在同域新增命名职责 | `NewExportService(...)`；无需把原有 `NewService` 改名 |
| 测试文件 | 模块前缀 + `_impl` + `_test` | `health_check_service_impl_test.go` |

包边界：`<module>` 目录 = 业务域包。一个包可以维护多个 controller/service/manager 与接口；拆新包的信号是**业务域变化**（独立路由前缀、依赖图、事务边界或生命周期），不是接口数量或路径数量。

同域协作者规则：

- Controller 的 `Register(api)` 可以注册本业务域的多个 path；每个 path 调用同一 Service 的不同方法即可。仅当上层调用者确实需要替换多种 Controller 实现时，才抽取窄接口。
- Service 的一个接口表达一组共享业务规则和依赖的用例；不同 HTTP path 不自动意味着新增 Service。
- Manager 的一个接口表达共享技术依赖的能力；不包含业务判断，也不以接口数量作为拆包依据。
- 当某个协作者有独立依赖、事务或生命周期，才新增同域职责和命名构造函数，例如 `NewExportService`；不要因第二个路由或第二个方法而全量改名。

- `server/service/<module>/interface.go`：Service 接口（供 mockgen 生成 mock）
- `server/service/<module>/<module>_service_impl.go`：默认 Service 实现；独立协作者再以职责命名
- `server/manager/<module>/`：技术能力层，同样采用「接口 + 实现」形态
- `server/controller/<module>/`：一个具体 Controller 可维护本域多个 HTTP 操作、DTO 与转换
- `server/wire/<module>/wire.go`：依赖组装（默认提供 `NewController`）

### 5. 注册路由

`server/router/router.go` 的 `registerV1GroupAPI` 中：

```go
func registerV1GroupAPI(api huma.API, middleware ...huma.Middlewares) {
	user.NewController().Register(api)   // 挂到 /<服务名>/api/v1/**
}
```

### swag 注解 → huma 对照

| swag（旧） | huma（新） |
|------------|------------|
| `@Summary` | `Operation.Summary` |
| `@Description` | `Operation.Description` |
| `@Tags` | `Operation.Tags` |
| `@ID` | `Operation.OperationID` |
| `@Accept` / `@Produce` | `Operation.ContentTypes`（默认 application/json） |
| `@Param` | 请求结构体字段 huma tag（`path`/`query`/`header`/`body`） |
| `@Success` | 出参结构体（`Body` 字段 + `humax.Envelope`） |
| `@Failure` | error + `huma.StatusError`（见 `common/humax.Error`） |
| `@Router` | `Operation.Method` + `Operation.Path` |

## 注意事项（写代码前必读）

### 路由与文档

- **前缀**：`huma.NewGroup(api, "/xxx")` 的前缀同时写入路由与 OpenAPI 文档 `paths`。
  文档用 `serverAPI.OpenAPI().Downgrade()` 生成（含前缀），knife4go 前端检测到
  paths 已带前缀后不再拼接，因此**展示路径 = 文档中的完整路径**。
- **knife4go 注册**：必须放在**所有路由注册之后**（文档才完整），且仅在调试模式
  （`gin.IsDebugging()`）下注册。
- **不要**手动改 `OpenAPI()` 生成的文档内容。

### 分层与依赖

- 控制器不写业务逻辑，服务不直连 DAO——调用链单向。
- 依赖注入通过 `wire/` 组装，控制器/服务/管理器构造时传入依赖，禁止全局单例硬编码。

### 错误与响应

- handler 返回的 error **必须携带状态码**（实现 `huma.StatusError` 的
  `GetStatus() int`），否则 huma 默认按 200/500 处理。
- 错误统一走 `common/humax.Error`（信封 + 状态码）；需要新状态码时仿照
  `InternalServerError` 增加构造函数。
- 成功响应统一 `humax.Envelope`（版本 + code/message/data）。

### 中间件

- **gin 中间件**（`server/middleware/`、potato 中间件）：作用于整个 gin.Engine，
  在 `InitApiRouter` 的 `rootMiddleware` 注册。
- **huma 中间件**（`huma.Middlewares`）：按组挂载（`huma.Group.UseMiddleware`），
  路由组注册函数（`registerBaseAPI` 等）的 `middleware` 参数预留了入口。

### 工具链

- **DAO/Model 生成**：`bash script/gentol.sh`（配置见 `conf/db/`，完整说明见
  [script/README.md](script/README.md)）。
- **接口 mock**：`bash script/go-mockgen.sh`（生成到 `mocks/`，接口文件在
  `server/service|manager/<module>/interface.go`）。
- **测试**：`go test ./...`；分层测试（controller/service/manager）使用 mock。
- **格式**：`gofmt` / `golangci-lint`；注释使用 GoDoc 风格，业务意图用中文。

### 文档 UI（Knife4j）

- 文档 UI 由 knife4go 提供：`/{服务名}/doc.html`、`/{服务名}/v3/api-docs`、
  `/{服务名}/v3/api-docs/swagger-config`。
- knife4go 的静态资产路由以 Hidden 操作注册，不会出现在 OpenAPI 文档中。
- 服务名（`application.name`）即 URL 前缀，修改配置即可调整全部接口前缀。

## 工具介绍（保留）

### 1、gentol 使用

项目通过统一脚本完成 DAO/Model 生成和 DDL 执行。若 `conf/db/<DB_CONF>`（默认 `db.toml`）存在，则只读取该 TOML 文件；否则才读取 `conf/application.yaml`。环境变量始终优先于被选中的配置文件，配置值后的空白加 `#` 注释会被忽略。

```shell
## 安装 gentol
go install github.com/jasonlabz/gentol@master

## 设置数据库环境变量
export DB_TYPE=postgres
export DB_HOST=127.0.0.1
export DB_PORT=5432
export DB_USER=postgres
export DB_PASS='your-password'
export DB_NAME=example
export DB_SCHEMA=public

## 生成 DAO/Model
bash script/gentol.sh

## 执行 DDL
bash script/gentol.sh ddl conf/migrations/20240701_001_example_add_column.sql
```

PowerShell 使用 `./script/gentol.ps1` 和 `./script/gentol.ps1 ddl <sql文件>`。完整环境变量和参数说明见 [script/README.md](script/README.md)。

### 2、API 文档

项目使用 Huma 生成 OpenAPI 3.0 文档，并使用 Knife4go 提供文档 UI。无需安装或运行额外的文档生成命令。

调试模式下，文档 UI 位于 `/{service}/doc.html`，生成的 OpenAPI 3.0 文档位于 `/{service}/v3/api-docs`。
