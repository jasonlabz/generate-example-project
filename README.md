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
├── apperr/                全局错误码目录（code + HTTP 语义 + 公开文案）
├── humax/                 huma 响应、分页、文件流与错误封装
└── consts/ helper/ resource/   常量、辅助函数与全局组件
server/
├── router/router.go       路由组装：huma 组层级 + 中间件 + knife4go 文档
├── controller/            控制器层：huma 路由注册 + DTO 定义 + 转换
├── service/               服务层：业务逻辑入口（用例编排）
├── manager/               技术能力层：外部系统/基础设施访问（DB、Redis 等）
├── middleware/            gin 中间件（上下文、日志）
└── wire/                  组合根：集中装配各模块对象图（不按模块拆目录）
mocks/                     mockgen 生成的接口 mock（bash script/go-mockgen.sh）
conf/                      配置（application.yaml、日志、迁移 SQL）
script/                    gentol（DAO/Model 生成）、go-mockgen 脚本
docs/                      设计文档、开发过程记录
```

> 各层都以 `health_check`（最小样例）与 `user`（完整样例：参数、分页、文件流、错误码）两个模块作参照，
> 新增业务模块直接照抄 `user` 的结构。

### 分层职责（强制）

| 层 | 职责 | 禁止 |
|----|------|------|
| controller | 路由注册、参数/响应模型、DTO 与业务模型转换 | 业务逻辑、直连 DB |
| service | 业务用例编排（调用 manager 组合能力） | 直接访问 DAO/外部系统 |
| manager | 单一技术能力（DB 访问、Redis、外部 API） | 业务判断 |
| wire | 依赖装配 | 逻辑 |

调用链：`HTTP → controller → service → manager → DAO/外部系统`

## Huma 编写规范（新增业务模块模板）

新增模块按以下步骤。文件结构参照 `server/*/user/`（完整样例：参数、分页、请求体、文件流、错误码），
`health_check` 是最小样例，只看结构即可：

### 1. 控制器：`server/controller/<module>/register.go`

```go
// Register 把模块的 HTTP 操作注册到 api（huma 组）。
// huma.Register 三要素：
//  1. api：huma.API 或 huma.NewGroup（组自动应用前缀）；
//  2. huma.Operation：OpenAPI 元信息（Summary/Tags/OperationID）；
//  3. handler：humax.Wrap 负责统一成功/错误信封。
func (c *Controller) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "user-get",
		Method:      http.MethodGet,
		Path:        "/users/{id}",        // 路径参数用 {name} 占位
		Summary:     "获取用户",
		Tags:        []string{"用户"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusInternalServerError},
	}, humax.Wrap(consts.APIVersionV1, c.handleGet))
}
```

### 2. 请求/响应模型：`types.go`

```go
// 请求参数结构体用 huma tag 声明位置与校验，huma 自动生成 OpenAPI 参数。
type getUserInput struct {
	ID int64 `path:"id" minimum:"1" example:"1"` // path/query/header/body 四类位置
}

// 成功响应由 humax.Wrap 自动封装为 Envelope[*userVO]，无需重复定义 output。
```

查询参数必须按业务语义区分：筛选条件不传表示“不限定”，不要加 `required`；只有接口
无法执行时才加 `required:"true"`。`minimum` / `maximum` 只校验已提供的值，不能代替必填。
分页列表统一采用第一页、每页 200 条的默认值，并限制最大页大小为 200：

```go
type listUsersInput struct {
	TenantID string `query:"tenant_id" required:"true" minLength:"1" doc:"租户标识"`
	Keyword  string `query:"keyword" doc:"姓名或账号筛选；不传不筛选"`
	Page     int64  `query:"page" default:"1" minimum:"1" doc:"页码"`
	PageSize int64  `query:"page_size" default:"200" minimum:"1" maximum:"200" doc:"每页条数"`
}
```

### 3. 处理器：`<module>_controller.go`

```go
// handleGet 只负责用例调用与 DTO 转换；协议错误由 Wrap 统一映射。
func (c *Controller) handleGet(ctx context.Context, in *getUserInput) (*userVO, error) {
	user, err := c.service.GetUser(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return toUserVO(user), nil
}
```

列表接口返回数据与分页信息，由 `humax.WrapPage` 统一封装：

```go
func (c *Controller) handleList(ctx context.Context, in *listUsersInput) ([]userVO, *humax.Pagination, error) {
	pagination := &humax.Pagination{Page: in.Page, PageSize: in.PageSize}
	items, total, err := c.service.List(ctx, in.Keyword, pagination.GetOffset(), in.PageSize)
	if err != nil {
		return nil, nil, err
	}
	pagination.Total = total
	pagination.GetPageCount()
	return toUserVOs(items), pagination, nil
}

// 注册时使用 humax.WrapPage(consts.APIVersionV1, c.handleList)。
```

`Operation.DefaultStatus` 固定为 `http.StatusOK`。`Operation.Errors` 声明该接口**实际可能返回**
的状态码，例如查询单条声明 `[]int{http.StatusNotFound, http.StatusInternalServerError}`：
业务错误会带上 `apperr` 目录中登记的 HTTP 语义（100004001 → 404），只有未预期的故障才是 500。
声明齐全才能让 OpenAPI 文档与真实行为一致，前端按文档生成的 client 才不会漏掉分支。

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
- `server/wire/<module>.go`：依赖组装，一个模块一个文件，导出 `New<Module>Controller`
  （如 `wire/user.go` → `NewUserController`）

### 5. 注册路由

`server/router/router.go` 的 `registerV1GroupAPI` 中：

```go
func registerV1GroupAPI(api huma.API, middleware ...huma.Middlewares) {
	wire.NewUserController().Register(api)   // 挂到 /<服务名>/api/v1/**
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
| `@Success` | `humax.Wrap` / `humax.WrapPage` 推导的出参（统一为 `humax.Envelope`） |
| `@Failure` | error + `humax.BusinessError`（见 `common/humax.Error`） |
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

响应信封统一为 `humax.Envelope`（版本 + code/message/data），错误响应复用同一结构，不退回 RFC7807。
HTTP 状态与业务 code 是**双轨**：HTTP 状态让调用方做通用处理，业务 code 让调用方精确定位。

**业务失败用真实的 4xx，不用 HTTP 200。** 两者都能表达失败，选 4xx 的理由：

| 维度 | HTTP 200 + 业务 code | 4xx + 业务 code（本模板） |
|------|---------------------|--------------------------|
| 成功率/SLO 指标 | 业务失败被算作成功，告警失真 | 4xx 与 5xx 天然分离 |
| 网关与客户端重试 | 无法按状态码决策 | 可对 5xx 重试、对 4xx 不重试 |
| OpenAPI 文档 | 只能描述 200，失败分支丢失 | 失败状态可如实声明 |
| 前端处理成本 | — | 拦截器已按 status 分流，code 用于细分 |

> 若上游系统强制要求"一律 200"，改 `apperr.Spec.HTTPStatus` 的取值即可，code 体系不受影响；
> 但请同步接受成功率指标失真的代价。

```text
成功            HTTP 200  code = 0
参数校验失败     HTTP 422  code = 100001001   （框架错误，huma 自动映射）
业务失败         HTTP 404/409/422/403/...      code = 目录中登记的业务码
未知内部错误     HTTP 500  code = 100008001   （对外恒定文案，不泄漏内部 cause）
```

> huma 用 **422**（不是 400）表示参数不合法——必填缺失、取值越界、格式错误都走这里。
> `apperr.ForHTTPStatus` 把框架 422 归到 `InvalidRequest`，同时保留 422 的 HTTP 语义。
> 否则它会和业务代码主动表达的 `BusinessRule`（同为 422）撞码，调用方无法区分
> "我参数写错了"和"业务规则不允许"。

- Controller handler 返回原始业务数据与 error，由 `humax.Wrap` / `humax.WrapPage` 统一加壳。
- **业务失败优先走 `apperr` 目录**（推荐）：service 层返回 `apperr.NotFound.WithMessage(...)`，
  controller 原样上抛，`humax` 自动映射为目录登记的 HTTP 状态 + code。
- Controller 层遇到"参数本身合法、但当前组合不被支持"的失败时，用
  `humax.BusinessError(version, code, message)`：它同样返回统一信封，不落入 RFC7807。
  未登记的 code 会回退为 `InvalidRequest` 的 HTTP 语义，但保留传入的 code 与 message。
- `humax.ConfigureHumaErrorFactory` 必须在创建 Huma API 前于 Router 中调用；参数校验
  错误因此返回 HTTP 422 + code 100001001（不是 200）。
- 未知内部错误对外使用稳定的 `服务内部错误`，不把 cause 写进 `message`。

#### message 与 err_trace 的分工

`message` 是**概括性文案**，给调用方直接展示；`err_trace` 是**详细报错链**，给开发者定位问题。

`humax.ConfigureErrorDetails(true)`（Router 中由 `gin.IsDebugging()` 驱动，即
`application.debug: true`）打开后，**任何错误都会带上 err_trace**，4xx / 5xx 一视同仁：

```go
// 5xx：cause 是未知故障，err_trace 含 fmt.Errorf 逐层包装的完整上下文
fmt.Errorf("query user: %w", dbErr)
// → message: "服务内部错误"        err_trace: "query user: connection refused: 127.0.0.1:5432"

// 4xx：cause 是 apperr 目录错误，err_trace 含目录 code 与内部原因
apperr.NotFound.WithErr(errors.New("db: timeout"))
// → message: "请求的资源不存在"     err_trace: "100004001 -> 请求的资源不存在\n inner error: db: timeout"

// 4xx：只有业务文案、没有内部原因时，回退为目录链，保证调试时看得到来源
apperr.NotFound.WithMessage("用户 9 不存在")
// → message: "用户 9 不存在"       err_trace: "100004001 -> 用户 9 不存在"

// controller 主动判定的业务失败
humax.BusinessError(consts.APIVersionV1, apperr.BusinessRule.Code(), "暂不支持 xlsx 导出")
// → message: "暂不支持 xlsx 导出"   err_trace: "100006001 -> 暂不支持 xlsx 导出"
```

关闭开关（生产环境）时 `err_trace` 字段整体省略（`omitempty`），内部原因不出现在响应中。

> 它会把表名、SQL、下游地址等内部细节暴露给调用方，**只能在可信环境开启**——
> 上线前务必确认 `application.debug: false`。
> 生产环境需要排查内部原因时，应写入服务端日志而非响应体。

**错误码全链路示范**（见 `server/service/user`、`server/controller/user`）：

```go
// service：把"记录不存在"这类可预期结果转换为目录中的错误码，而不是 fmt.Errorf 丢掉语义。
record, exists, err := s.manager.FindByID(ctx, id)
if err != nil {
    return User{}, fmt.Errorf("find user %d: %w", id, err)   // 技术故障 → 500
}
if !exists {
    return User{}, apperr.NotFound.WithMessage(fmt.Sprintf("用户 %d 不存在", id))  // 业务失败 → 404
}

// controller：原样上抛，不做任何转换或包装，错误码才不会在传递中丢失。
func (c *Controller) handleGet(ctx context.Context, in *getUserInput) (*userVO, error) {
    item, err := c.service.Get(ctx, in.ID)
    if err != nil {
        return nil, err
    }
    view := toUserVO(item)
    return &view, nil
}

// controller：协议层面的业务失败用 BusinessError（参数合法但组合不支持）。
if in.Format == "xlsx" {
    return nil, humax.BusinessError(consts.APIVersionV1, apperr.BusinessRule.Code(), "暂不支持 xlsx 导出")
}
```

> **关键点：`%w` 与 `%v` 的区别**。`humax.FromError` 用 `errors.As` 在整条错误链里找
> `potato/errors.IError`，所以 `fmt.Errorf("...: %w", apperrErr)` 包装后 code 和 HTTP 语义
> **不会丢**（实测包两层仍然正确映射）。真正会丢的是 `%v`——它把错误降级成字符串，
> 链断掉之后只能落回 500。
>
> ```go
> fmt.Errorf("get user: %w", apperr.NotFound.WithErr(dbErr))  // → 404 + 100004001 ✅
> fmt.Errorf("get user: %v", apperr.NotFound.WithErr(dbErr))  // → 500 + 100008001 ❌
> ```
>
> 结论：包装业务错误时坚持用 `%w` 即可，不必为了避免丢码而放弃包装上下文。

#### 错误链的服务端记录

响应里的 `err_trace` 只在 debug 模式出现，生产环境必须靠日志排查。`humax` 通过
`SetErrorReporter` 把 handler 返回的**原始错误**交出来（响应体已脱敏、`gin.Errors` 又恒为空，
这是唯一的时机）：

```go
// server/router/router.go
humax.SetErrorReporter(middleware.LogErrorChain)
```

`LogErrorChain` 只记录 5xx（4xx 是可预期的业务失败，逐条记录会淹没故障信号），
日志携带请求上下文，可自动关联 `trace_id`：

```text
ERROR	middleware/huma.go:301	[humax] internal error (status=500): query user: connection refused: 127.0.0.1:5432
```

> 不需要强制要求"返回给 huma 的 error 必须用 potato error 包住"：非 `IError` 的错误会被
> 安全地兜底成 500（对外恒定文案、不泄漏 cause），而错误链始终会被 `ErrorReporter` 完整记录。
> 强制包装只会增加形式负担，换不来额外保障。

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

完整环境变量和参数说明见 [script/README.md](script/README.md)。

### 2、API 文档

项目使用 Huma 生成 OpenAPI 3.0 文档，并使用 Knife4go 提供文档 UI。无需安装或运行额外的文档生成命令。

调试模式下，文档 UI 位于 `/{service}/doc.html`，生成的 OpenAPI 3.0 文档位于 `/{service}/v3/api-docs`。
