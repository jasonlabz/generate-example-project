# generate-example-project 优化分析

> 目标：把本项目打造成**其他 Web 后端项目的统一基准模板**。
> 因此评判标准不是"能不能跑"，而是"新项目照抄时会不会踩坑、会不会学错"。

## 0. 验证方式与当前状态

### 2026-09-19 复测（修复后）

| 检查项 | 命令 | 结果 |
|--------|------|------|
| 编译 | `go build ./...` | ✅ 通过（go.sum 已补齐，go.mod 同步升级） |
| 全量测试 | `go test ./...` | ✅ 9 个包全部 ok（P0-2 已修） |
| 外部依赖泄漏 | `grep dagine go.mod` | ✅ 已移除，go.mod 只剩本项目应依赖的模块 |
| 测试耗时 | `go test ./...` | ⚠️ `script` 包 ~62s，占整体 95% |
| 端到端冒烟 | 真实起服务 curl | ✅ 见下方"冒烟验证" |

### 冒烟验证（真实起服务后 curl，2026-09-19）

| 请求 | 响应 |
|------|------|
| `GET /health-check` | `200` `code=0` `data=["success"]` |
| `GET /generate-example-project/api/v1/users` | `200` 带 `pagination{page:1,page_size:200,page_count:1,total:2}` |
| `GET .../users/9999` | **`404`** `code=100004001` `message="用户 9999 不存在"` |
| `POST .../users`（新用户） | `200` `data={id:3,name:"carol"}` |
| `POST .../users`（重名） | **`409`** `code=100005001` `message="用户名 carol 已存在"` |
| `GET .../users/export?format=csv` | `200` `Content-Type: text/csv`，三行数据 |
| `GET .../users/export?format=xlsx` | **`422`** `code=100006001`（BusinessError 生效） |

这组结果说明错误码全链路已打通：service 抛出的 `apperr` 码一路传到 HTTP 状态与业务 code，
没有被 `fmt.Errorf` 吃掉、没有降级成 500。

**结论：编译、测试、冒烟链路全部打通；第一阶段 7 项已全部完成。**

---

## P0. 阻塞项 —— 必须立即修，否则"照抄即崩"

> 状态：P0-1 ✅ 已修（2026-09-19）；P0-2 ✅ 已修（2026-09-19）；P0-3 ❌ 未修。

### P0-1 `go.sum` 缺哈希，全新 clone 无法编译 ✅ 已修

`go.mod` 声明 `github.com/jasonlabz/potato v1.0.12`，但 `go.sum` 只有 `/go.mod` 哈希、缺 `h1:` 哈希：

```
common\resource\resource.go:4:2: missing go.sum entry for module providing package github.com/jasonlabz/potato/es
```

**影响**：新同学 clone 后 `go build` / `go test` 全线失败，且报错信息完全指向不到根因。
**修复**：`go mod tidy` 后提交完整 go.sum；在 CI 加一条 `go mod verify` + `-mod=readonly` 构建。

### P0-2 `server/middleware/huma_test.go` 编译失败（跨项目残留）✅ 已修

> 注意：中途曾出现一种**错误的修法**——为了让 `dagine-dashboard` 的 import 能解析，把它加进 `go.mod` 的
> `require` 块当直接依赖。这会把"测试引用了外部项目"从编译错误变成永久依赖，
> 所有下游项目 clone 模板时都会被拖进一个无关仓库。正确做法是改回本项目的 `common/resource`。

两个独立问题叠加：

1. 第 15 行 import 了**另一个项目**的包：
   ```go
   "github.com/jasonlabz/dagine-dashboard/common/resource"   // 应为本项目 common/resource
   ```
2. 第 185 行调用了**不存在的函数**：
   ```go
   api.UseMiddleware(HumaRequestMiddleware(WithHumaSkipHiddenOperations()))
   ```
   `server/middleware/huma.go` 中既没有 `WithHumaSkipHiddenOperations`，`HumaRequestMiddleware` 也不接受可选参数。

**影响**：整个 middleware 包测试跑不起来，`go test ./...` 直接 FAIL。这是最扎眼的一处——基准模板自己的测试是红的。

**已采用的修复**（2026-09-19）：
1. `huma_test.go:15` import 改回 `github.com/jasonlabz/generate-example-project/common/resource`；
2. `go mod tidy` 移除 `dagine-dashboard` 直接依赖；
3. 在 `server/middleware/huma.go` **补上缺失的能力**而非删用例：
   - `HumaOptions` 增加 `skipHidden` 字段；
   - 新增 `WithHumaSkipHiddenOperations()`；
   - `HumaRequestMiddleware(opts ...HumaOption)` 支持可变参数，命中 `ctx.Operation().Hidden` 时直接放行。

   选择补实现而不是删用例的理由：knife4go 的 `doc.html` / 静态资产是以 Hidden 操作注册的，
   为它们记录请求日志、替换响应 Writer 捕获 body 本就不合理——这个能力是有实际价值的，
   测试文件里留着它说明原设计就有此意图，只是复制时漏了实现。

### P0-3 `web/` 是空壳，前端链路全线断裂

`git ls-files | grep web` 只有 `web/README.md`。目录里没有 `package.json`、`vite.config.*`、`src/`。
但以下三处都依赖它：

| 依赖方 | 位置 | 现状 |
|--------|------|------|
| `make frontend` | Makefile:37 | `cd web && pnpm install` → 无 package.json，失败 |
| `make package` | Makefile:53/61 | 依赖 `copy-frontend` → 连带失败 |
| `docker build` | Dockerfile:8-11 | `COPY web/package.json web/pnpm-lock.yaml* ./` → 失败 |

**修复**（二选一）：
- 提供一份最小可构建的前端（Vue3 + Vite + `pnpm build` → `web/dist`），让"前后端不分离部署"这条链路真的成立；
- 或者彻底移除前端：删 `web/`、Makefile 的 `frontend`/`copy-frontend`、Dockerfile Stage 1、`docker-build.sh` 里的 Web 访问地址。

**建议选后者**——这是**后端**基准模板，塞一个不跑的前端只会增加维护负担和误解。

**已采用后者**（2026-09-19）：`web/` 目录连同 `web/README.md` 一并删除；Makefile 移除
`frontend` / `copy-frontend` 目标与 `webroot` 相关逻辑；Dockerfile 只保留后端两阶段构建；
`.gitignore` / `.dockerignore` / `docker-build.sh` 中的 web 条目清理干净。

---

## P1. 文档与代码漂移 —— 模板最致命的一类问题

模板的 README 就是"教程"。教程教的代码编译不过，比没有教程更糟。

### P1-1 `humax.BusinessError`  pervasive 引用但根本不存在

| 引用位置 | 内容 |
|----------|------|
| README.md:177 | `@Failure` → `humax.BusinessError` |
| README.md:199-201 | "Controller 返回 `humax.BusinessError(version, code, message)`，返回 HTTP 200 与非零业务 code" |
| server/README.md:48 | "预期错误返回 `common/humax.BusinessError`" |
| common/README.md:23 | 同上 |
| common/humax/README.md:14,29 | 给出完整调用示例 |

但 `common/humax/response.go` 里**没有这个函数**。现有能力只有 `FromError` / `MapError` / `InternalServerError` / `newCatalogError`。

**修复**：补一个
```go
// BusinessError 构造预期内的业务失败：HTTP 200 + 非零 code，不落入 RFC7807。
func BusinessError(version string, code int, message string) error
```
并让它能被 `FromError` 原样保留。这是整个错误契约的关键一环。

### P1-2 错误码体系形同虚设

`common/apperr/errors.go` 定义了 9 个错误码（InvalidRequest / Unauthenticated / Forbidden / NotFound / Conflict / BusinessRule / DependencyUnavailable / Internal / RateLimited），但：

- `grep apperr\.` 结果显示**只有 `common/humax` 内部在用**，业务代码零引用；
- `service` 层一律 `fmt.Errorf("check health: %w", err)`，错误码在 wrap 过程中丢失；
- `humax.FromError` 只识别 `potatoErrors.IError`，普通 `fmt.Errorf` 一律降级成 500。

**后果**：照抄模板的项目会发现"我返回了 NotFound，前端却收到 500"，然后各自造轮子。
**修复**：在示例模块里完整示范一次——service 返回 `apperr.NotFound.WithErr(err)`（或 `WithMessage`），controller 原样上抛，`humax` 映射成 HTTP 404 + code=100004001。并把这个链路写进 README 的"新增模块模板"第 3 步。

### P1-3 README 描述的目录有 1/3 不存在

| README 声明 | 实际 |
|-------------|------|
| `common/ginx/`（gin Context 写入适配） | ❌ 不存在 |
| `docs/`（设计文档、开发过程记录） | ❌ 不存在（本文件即为其补齐） |
| `server/README.md` 提到的 `dal/db/dao/` | ❌ 不存在 |
| `conf/seed/`、`conf/migrations/` | ✅ 存在 |

**修复**：删除不存在的条目，或补齐实现。模板的目录树必须是可核对的。

### P1-4 静态服务"配置有、实现无"

- `conf/application.yaml` 有 `application.server.static.enable/port/path`；
- `bootstrap/config.go:158` 有 `StaticConfig` 类型，但 `ServerConfig` 里**没有该字段**，配置被静默丢弃；
- `main.go` 完全没有启动静态服务；
- README 开头却写着"`application.server.static.enable` 默认 `false`，需显式配置为 `true` 才会启动"；
- `docker-build.sh:87` 甚至印出 `Web: http://localhost:8080/server/`，而 `router_test.go:210` 专门断言 `/server/` 必须 404。

**修复**：要么实现（静态文件服务 + 接到 `ServerConfig`），要么从配置、README、Dockerfile、docker-build.sh 里全部删掉。

### P1-7 框架参数校验与业务规则错误撞码 ✅ 已修（2026-09-19）

实测发现 huma 用 **422**（不是 400）表示参数校验失败，而 `ForHTTPStatus(422)` 返回 `BusinessRule`，
于是两类完全不同的失败共用同一个 code：

| 场景 | 修复前 | 修复后 |
|------|--------|--------|
| `page_size=999`（框架参数校验） | 422 + **100006001** | 422 + **100001001** |
| `format=xlsx`（业务规则） | 422 + **100006001** | 422 + **100006001** |

**后果**：调用方无法区分"我参数写错了"与"业务规则不允许"，只能去解析 message 文本。

**修复**：`apperr.ForHTTPStatus` 的 422 分支改为返回 `Spec{HTTPStatus: 422, base: InvalidRequest.base}`——
归因到 `InvalidRequest`，同时保留 huma 的 422 HTTP 语义。`BusinessRule` 保持自己的 code 不变。
`detailTrace` / `detailCause` 见 P3-7b。

### P1-5 Dockerfile 环境变量注释是另一个项目的

Dockerfile:61-72 注释写的是 `DAGINE_DB_HOST` / `DAGINE_JWT_SECRET` / `DAGINE_STATIC_PATH`……
本项目是 `generate-example-project`，且实际并未实现任何环境变量覆盖。**这是典型的复制粘贴残留，对"基准模板"的公信力杀伤最大。**

### P1-6 配置结构体与 YAML 不同步

`conf/application.yaml` 有 `rocketmq:` 和 `obs:` 两段，`Config` 结构体里**没有对应字段** → 静默忽略，排查时会非常困惑。
另外 `Config.Validate()`（config.go:316）定义了但**从未被调用**，配置的端口/服务名非法时不会 fail-fast。

---

## P2. 示范性缺失 —— 作为"模板"最该有但没有的

这部分决定了别的项目"能不能照着抄出完整业务"，比修 bug 更重要。

### P2-1 缺少一个完整的 CRUD 业务模块（最高优先级）

现状：`registerV1GroupAPI` 是**空函数**，全项目唯一的业务模块是 `health_check`，而且它：
- 注册在 `rooterAPI`（根级，无 `/<服务名>` 前缀），与 README "业务路由统一挂 `/<服务名>`" 矛盾；
- handler 签名是 `func(ctx, _ *struct{})`，**没有任何输入参数**；
- 没有 DTO、没有分页、没有错误分支。

**后果**：开发者抄模板时，path 参数怎么写、query 怎么写、body 怎么写、分页怎么接、错误怎么抛——全靠读 README 想象。

**修复建议**：新增一个 `user`（或 `example`）模块，在 `/<服务名>/api/v1` 下示范齐全：

```
GET    /users/{id}              路径参数 + 404 错误码
GET    /users                   query 筛选 + humax.WrapPage 分页（page=1, page_size=200, max 200）
POST   /users                   body + 必填校验 + 409 冲突
DELETE /users/{id}              204/200 + 幂等
GET    /users/export            文件下载（humax.File / StreamResponse）
```

一次把 README 里所有"注意事项"落成可运行代码。

### P2-2 没有数据访问层（DAO）示范

`server/README.md` 提到 `dal/db/dao/`，`script/gentol.sh` 也能生成 DAO/Model，但仓库里**没有任何调用 DAO 的代码**。
而"Web 后端模板"最核心的一问就是：**service/manager 怎么访问数据库、事务边界在哪、读写分离怎么接**。

**修复建议**：
- 用 gentol 生成一份最小 model/dao（比如 `example` 表，`conf/migrations` 里已有 baseline）；
- manager 层示范事务（`db.Transaction`）+ 上下文超时；
- 明确"事务边界在 service 还是 manager"并写进 README（现在 README 只说"manager 单一技术能力"，没说事务归谁）。

### P2-3 没有认证/授权接入示例

`apperr` 里定义了 `Unauthenticated` / `Forbidden`，但没有任何中间件或服务示范如何校验 token、如何把 userID 注入 context（`common/helper` 有 `GetUserID` 但没人调用）。

### P2-4 没有限流 / 幂等 / 重试示范

`apperr.RateLimited` 已定义但无实现。后端服务这三个几乎是刚需。

### P2-5 没有版本演进示例

`router.go:103-104` 的 v2 组是注释掉的。建议保留注释但补一段说明"新增版本时的完整步骤"，或者直接给出 `/v2` 的空骨架。

### P2-6 缺 OpenAPI 产物导出

只有运行时 `/v3/api-docs`。建议提供一个 `make openapi` 把 OpenAPI 3.0 JSON 落到 `docs/openapi.json`，供前端生成 TS client、供 CI 做契约 diff 检测。

---

## P3. 工程质量与交付链路

### P3-1 没有 CI 配置

仓库没有 `.github/workflows`、` .gitlab-ci.yml` 或任何流水线文件。基准模板应该自带：

```yaml
- go build -mod=readonly ./...      # 同时挡住 P0-1
- go vet ./...
- golangci-lint run                 # 配置已写好却没人跑
- go test -race -cover ./...
- 覆盖率门禁（建议 ≥ 60%，common/ 和 humax 应 ≥ 80%）
```

### P3-1b 换行符混用（LF / CRLF）

`gofmt -l` 会报出 `common/`、`bootstrap/`、`cmd/` 下的一批文件，但 `gofmt -d` 显示差异**只有 `^M`**，
即这些文件是 CRLF，而 `server/` 下是 LF——仓库内部换行符不统一。

**影响**：`gofmt -l` 在 CI 里会误报失败；跨平台 diff 噪音大。
**修复**：新增 `.gitattributes` 固定 `*.go text eol=lf`，并把存量文件 normalize 一次。

### P3-2 Makefile 缺关键 target

现有 `prepare/build/test/package/clean/frontend/copy-frontend/docker`。
建议补：`lint`、`fmt`、`vet`、`tidy`、`openapi`、`help`（默认 target 打印帮助）。

### P3-3 `script` 包测试拖慢整体

`go test ./...` 里 `script` 包耗时 **60s**（要 `go build` 一个 fake mockgen + 起 Git Bash 子进程），占整体 95%。
建议加 `//go:build !short` 或 `testing.Short()` 跳过，让 `make test` 默认快、CI 全量跑。

### P3-4 `main.go` 的 goroutine 里用 `log.Fatalf`

```go
go func() {
    if err := srv.ListenAndServe(); ... {
        log.Fatalf("listen: %s\n", err)   // os.Exit(1)，跳过所有优雅退出
    }
}()
```
应改为把错误送回主流程统一 shutdown。pprof server 同样问题。

### P3-5 pprof 双份注册 + 端口冲突隐患

`startPProfServer` 同时做了两件事：
- 把 `/debug/pprof/*any` 挂到**主 gin router**（即业务 HTTP 端口 8080）；
- 又起了一个独立 pprof server（`pprofConf.Port`，默认配置也是 **8080**）。

默认 `application.yaml` 里 `http.port: 8080` 与 `pprof.port: 8080` 相同 → 一旦 enable 就冲突。
另外把 pprof 暴露在业务端口是安全隐患。建议只保留独立端口，并改默认端口为 6060 之类。

### P3-6 Dockerfile 基线偏旧 / 版本超前

- `golang:1.26-alpine3.23`：go.mod 是 `go 1.25.0`，用 1.26 镜像属于超前且与 `.golangci.yml` 的 `go: "1.25"` 不一致；
- `debian:bullseye-slim` 已进入 EOL 维护期，建议 `bookworm-slim`；
- `CGO_ENABLED=1` 是因为 sqlite（glebrez/go-sqlite 或 jasonlabz/sqlite），建议注释写明原因。

### P3-7 敏感配置硬编码入库

`conf/application.yaml` 里：
```yaml
crypto:
  - type: aes
    key: "0123456789abcdef"    # 硬编码密钥
  - type: des
    key: "des-key8"
redis:
  password: "******"
```
作为"典范"，应示范**环境变量覆盖 / 占位符 + 启动校验**的写法，而不是把密钥写进模板让人复制。同时 `gosec` 已启用，硬编码密钥是它重点扫的对象。

### P3-7b 500 的内部原因在生产环境彻底丢失 ✅ 已修（2026-09-19）

实测发现：handler 返回 error 时 `gin.Errors` 恒为空（`humagin` 适配器从不调用 `gin.Error()`）：

```
handler返回error后 -> gin.Errors = "" (len=0), status=500
```

而 `HumaRequestMiddleware` 的 `error_message` 字段正是取自它：

```go
errorMessage := ginContext.Errors.ByType(gin.ErrorTypePrivate).String()   // 恒为 ""
```

**后果**：`debug: false` 时，500 的内部原因既不返回给客户端（正确），也不出现在服务端日志
（缺陷）——线上只能看到"服务内部错误"，无法定位是哪个 SQL / 下游挂的。

**已做的缓解**：`message` 与 `err_trace` 明确分工——`message` 概括、`err_trace` 给完整报错链。
`ConfigureErrorDetails(true)` 打开后**任何错误都带 `err_trace`**（4xx / 5xx 一视同仁，不回退省略），
由 `detailTrace` 实现：有 `cause` 输出其完整 `Error()`，没有则回退到目录错误本身。

**已修复**（2026-09-19）：`humax` 新增 `SetErrorReporter` / `ReportError`，在错误离开 handler
的时机把**原始错误链**交给上层；Router 注册 `middleware.LogErrorChain`，只记 5xx，日志自动
关联 trace_id：

```
ERROR	middleware/huma.go:301	[humax] internal error (status=500): query user: connection refused: 127.0.0.1:5432
```

之所以要在 humax 层开口子：中间件拿不到 handler 的 error（`gin.Errors` 恒空），
响应体又已脱敏——错误离开 handler 的那一刻是唯一能拿到完整链的时机。
文件流接口不走 `Wrap`，需在 handler 里显式 `humax.ReportError(ctx, err)`。

### P3-8 自动迁移无开关

`bootstrap.MustInit` 里 `runMigrations` 无条件执行，只要 `datasource.enable: true` 就跑。
生产多实例场景虽然有分布式锁（`migrate_lock.go` 写得很扎实），但仍应提供 `datasource.auto_migrate: false` 开关，让生产走 `cmd/migrate` 独立发布。

---

## P4. 代码细节问题

| # | 位置 | 问题 |
|---|------|------|
| 4-1 | `server/router/router.go:122` | `registerRootAPI` 的注释是英文、结尾多了个 `\`，且描述与实现不符（它注册的是根级路由，不是 "server API"）。全项目注释风格应统一为中文 |
| 4-2 | `server/router/router.go:130` | `registerBaseAPI` 是空函数，注释却写"参数 middleware 为 huma 组中间件"，但签名里没有该参数 |
| 4-3 | `bootstrap/bootstrap.go:51` | `initResource` 空函数，注释 "all global variable should be initial" 语法错误且无信息量 |
| 4-4 | `common/helper/context.go:19-29` | `GetClientIP/GetUserID/GetToken` 是导出函数但**无 GoDoc 注释**；`.golangci.yml` 启用了 `revive.exported`，这说明 lint 从未真正跑过 |
| 4-5 | `common/humax/response.go:1` | 包注释用英文，与"业务意图用中文"的约定不一致 |
| 4-6 | `common/humax/response.go:432-453` | `FileResult` / `FileResultWithError` / `SimpleFileDownload` 与 `File` / `FileWithError` / `SimpleFile` **完全重复**。模板里两个等价 API 会让开发者纠结 |
| 4-7 | `common/humax/response.go:113` | `WrapList` 与 `Wrap` 职责重叠，建议明确取舍 |
| 4-8 | `common/humax/response.go:41` | `CurrentTime` 用 `time.Now().Format(time.DateTime)`（本地时间、无时区）。跨时区部署时建议 RFC3339 + 明确时区 |
| 4-9 | `common/humax/response.go:85` | `zeroIfNil` 对每个响应做反射，热路径有损耗；建议限定只在 slice/map/ptr 上走，或提供无反射版本 |
| 4-10 | `server/middleware/huma.go:84` | `HumaCORS()` 定义了但 router 里没用；且实现是 `Access-Control-Allow-Origin: <反射 Origin>` + `Allow-Credentials: true`，这是典型的不安全配置，作为模板示范应改为白名单 |
| 4-11 | `main.go:76` | `signal.Notify(quit, os.Interrupt, syscall.SIGINT, ...)` —— `os.Interrupt` 就是 `SIGINT`，重复注册 |
| 4-12 | `main.go` | 除 `startHTTPServer` 外，`startPProfServer`/`startGRPCServer` 无函数注释，与"函数必须有意图注释"的规范不符 |
| 4-13 | `conf/log/service.yaml` | `log_level: debug` + `write_file: true` 作为模板默认值偏重，建议 `info` |

---

## 5. 建议的落地顺序

### 第一阶段：让模板"可信"（1 天内）

1. ✅ `go mod tidy` 补 go.sum，CI 加 `-mod=readonly` 构建 + `go mod verify`
2. ✅ 修 `huma_test.go`：改回本项目 `common/resource` + 补 `WithHumaSkipHiddenOperations`
3. ✅ 补齐 `humax.BusinessError`，并新增 `user` 示例模块形成全链路闭环（2026-09-19）
4. ✅ 解除 `web/` 依赖：Dockerfile 去掉前端 Stage，`package` 不再依赖 `copy-frontend`（2026-09-19）
5. ✅ 修正 README 目录树与错误契约（2026-09-19）
6. ✅ 清理 Dockerfile 的 `DAGINE_*` 残留与 `/server/` 访问地址（2026-09-19）
7. ✅ 加 `.gitattributes` 统一 LF（2026-09-19）
8. ✅ `message` / `err_trace` 分工明确：debug 下任何错误都带完整报错链（P3-7b）
9. ✅ 修复框架参数校验与业务规则撞码（P1-7）
10. ✅ `wire` 层拍平并改为按模块拆文件（按用户设计要求）：子目录取消，
    每个模块一个文件，导出 `New<Module>Controller`
11. ✅ 构建体系收敛为「仅 Linux + 仅后端」：移除 Windows 分支与前端链路（见下方平台约定）

### 结构约定：wire 按模块拆文件，不拆目录

```text
server/
├── controller/<module>/    # 业务域 = 包边界，按模块建目录
├── service/<module>/
├── manager/<module>/
└── wire/                   # 组合根：单一包，一个模块一个文件
    ├── doc.go
    ├── health_check.go     # NewHealthCheckController
    └── user.go             # NewUserController
```

理由：目录层级只用于区分**职责**，业务边界由包名表达。在 wire 下再复制一层模块目录，
等于把同一个业务边界声明两遍，却不带来任何隔离收益（wire 本身就是汇聚点，天然需要看到所有模块）。

拆文件而非拆目录还有个实际收益：每个模块的文件只需导入自己那三层的包。拍成单文件时
controller/service/manager 的 6 个别名会交替出现（`healthcheckcontroller`、`usercontroller`、
`healthcheckmanager`……因为 gofmt 强制按导入路径字母序排列，无法按层分组）；
按模块拆文件后，一个文件里只有同一模块的三个别名，阅读是连贯的。

### 平台约定：仅支持 Linux + 仅后端

按用户要求收敛，不再维护第二套构建路径：

| 项 | 处理 |
|----|------|
| `Makefile` | 删除 `ifeq ($(OS),Windows_NT)` 全部分支（5 处）、`.exe` 后缀、`xcopy`/`rmdir` |
| `Dockerfile` | 两阶段均为 debian 系，声明仅 `linux/amd64`、`linux/arm64` |
| `script/gentol.ps1` | 删除 |
| `script/go-mockgen.sh` | 删除 `cygpath` / Windows 路径分支 |
| `script/go_mockgen_test.go` | 删除 Git Bash 平台探测（`gitBashPath` → `bashPath`），直接从 `PATH` 查 bash |
| `.gitattributes` | 删除 `*.ps1` / `*.bat` / `*.cmd` 的 CRLF 例外，全仓库统一 LF |
| `web/` | 删除（本模板只含后端） |

保留对**类 Unix 开发环境**的兼容（Linux / macOS / WSL / Git Bash）：Makefile 只用
`rm` / `cp` / `mkdir`，不依赖 GNU 专有参数，因此在 Windows 上通过 Git Bash 或 WSL
依然可以构建——只是不再为 `cmd.exe` 单独维护一套命令。

由于 controller / service / manager 三层的包名与业务域同名（都叫 `user`），
import 时统一用 `<module><layer>` 别名（`usercontroller` / `userservice` / `usermanager`）避免歧义。

### 第二阶段：让模板"够用"（1 周内）

7. 新增完整 CRUD 示例模块（路径/查询/body/分页/文件下载 + 全套错误码）
8. 接一条真实的数据访问链路（gentol 生成 model/dao + manager 事务示范）
9. service 层示范 `apperr` 错误码的正确抛出与透传
10. 补 CI（build / vet / lint / test / 覆盖率门禁）+ Makefile 的 `lint`/`fmt`/`vet`/`tidy`/`help`

### 第三阶段：让模板"先进"（2-4 周）

11. 认证/授权中间件 + userID 注入示范
12. 限流、幂等、重试三件套
13. 静态服务要么实现要么彻底移除（含 `StaticConfig` 接线）
14. 敏感配置走环境变量 + 启动校验
15. `make openapi` 导出契约 + CI 契约 diff
16. 统一注释语言、清理重复 API（`File*` 系列、`WrapList`）、修 `main.go` 的 `log.Fatalf`/pprof/信号重复

---

## 6. 值得肯定的部分（改动时请保留）

- **分层边界清晰**：controller / service / manager / wire 四层职责在 README 与代码里一致，`var _ Service = (*serviceImpl)(nil)` 这类编译期断言是好习惯。
- **错误信封设计正确**：`humax.ConfigureHumaErrorFactory` 在创建 huma API 前统一接管，避免 RFC7807 泄漏；`err_trace` 仅在 debug 暴露、内部错误对外恒定文案，安全边界考虑到位。
- **迁移锁实现扎实**：`bootstrap/migrate_lock.go` 区分原生咨询锁（pg/mysql/sqlserver）与表级兜底锁，独占 `*sql.Conn` 保证会话一致，陈旧锁抢占带条件删除——这块质量高于一般模板。
- **测试有分层**：controller 用 gomock 打 service，router 测 OpenAPI 契约与文档端点开关，humax 测错误映射——结构是对的，只是 middleware 包坏了。
- **`.golangci.yml` 写得非常用心**：每条 linter 都有启用/不启用理由注释。可惜没接进 CI，等于没生效。
