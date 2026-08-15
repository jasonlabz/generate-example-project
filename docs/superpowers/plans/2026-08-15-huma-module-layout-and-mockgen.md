# Huma 分层模块与 Mock 生成 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Health Check 重构为按业务模块组织、由 Wire 装配、可用 GoMock 隔离测试的 Huma 分层示例，并提供通用 Mock 生成入口。

**Architecture:** Controller、Service、Manager 的公开契约、类型和实现各自收敛在 `health_check` 业务目录；Wire 是唯一连接具体实现的组合根。Router 只取得 Wire 返回的 Controller 并注册路由。Mock 生成脚本使用一个私有 Go AST 扫描器识别接口，再以 mockgen source mode 生成与源码路径对应的测试替身。

**Tech Stack:** Go 1.25、Gin、Huma v2、GoMock v0.6.0、Bash、Go `go/ast`/`go/parser`。

## Global Constraints

- 直接在 `master` 工作区修改；不得创建 worktree。
- 用户明确要求本次不执行 `git commit`、`git push`、暂存重置或历史回退。
- 仅删除 `server/router/router.go` 中的 `staticRouter`；不得修改 Makefile、Dockerfile、构建脚本或 `webroot` 资源。
- Huma 的 `DocsPath`、`OpenAPIPath`、`SchemasPath` 和 `CreateHooks` 配置保持现状，保护 Knife4go 文档入口与既有 HTTP 响应形状。
- 所有新增导出 Go 标识符必须有 GoDoc；Mock 生成文件不得手工修改。
- `script/go-mockgen.sh` 固定调用 `go.uber.org/mock/mockgen@v0.6.0`，不依赖全局安装的 mockgen。
- Windows 上使用 Git Bash 执行 `.sh`；测试不得调用未配置 WSL 发行版的系统 `bash.exe`。

---

## Target File Structure

| 路径 | 最终职责 |
| --- | --- |
| `script/go-mockgen.sh` | 项目级 Mock 生成命令，接收 `-o` 输出根目录。 |
| `script/internal/mockscan/main.go` | 只供脚本调用的 AST 扫描器，输出含导出接口的源文件。 |
| `script/go_mockgen_test.go` | 以真实项目根目录验证脚本的常规、排除与 internal 路径规则。 |
| `server/manager/health_check/{interface.go,manager.go,local_probe.go}` | Manager 契约与实现。 |
| `server/service/health_check/{interface.go,types.go,service.go}` | Service 契约、结果类型与实现。 |
| `server/wire/health_check/wire.go` | Health Check 对象图装配。 |
| `server/mocks/server/<source-relative-dir>/mock_*.go` | 常规源码的生成 Mock。 |
| `<internal-parent>/.mocks/internal/<suffix>/mock_*.go` | `internal/` 源码的合法生成 Mock。 |

删除最终不再有职责的旧根契约、`*_impl.go`、空 `dto.go`/`helper.go`、扁平 Mock 文件及旧 `go:generate` 声明。

### Task 1: 建立可重复运行的通用 Mock 生成器

**Files:**
- Create: `script/internal/mockscan/main.go`
- Create: `script/go-mockgen.sh`
- Create: `script/go_mockgen_test.go`
- Create: `server/mocks/README.md`
- Modify: `script/README.md`
- Delete: `server/mocks/generate.go`

**Interfaces:**
- Consumes: `go.uber.org/mock/mockgen@v0.6.0`、项目根目录、`-o <output-root>`。
- Produces: `bash script/go-mockgen.sh -o server/mocks`；每个含顶层导出 interface 的源文件生成一个 `mocks` 包文件。

- [ ] **Step 1: 写出生成路径的失败测试**

在 `script/go_mockgen_test.go` 创建 `TestGoMockgen_GeneratesMirroredMocksAndSkipsOutput`。测试在项目根下临时创建 `cmd/mockgenfixture/internal/sample/contract.go`：

```go
package sample

import "context"

type Contract interface {
	Run(context.Context) error
}
```

测试定位 Git Bash 后，以 `script/go-mockgen.sh -o server/.test-mocks` 连续执行两次；断言存在：

```text
server/.test-mocks/server/controller/health_check/mock_register.go
cmd/mockgenfixture/.mocks/internal/sample/mock_contract.go
```

第一次运行后在 `server/.test-mocks/poison.go` 写入 `type MustNotGenerate interface { Stop() }`；第二次运行后断言不存在 `server/.test-mocks/server/.test-mocks/mock_poison.go`。用 `t.Cleanup` 删除测试输出、临时 fixture 和 `.mocks` 目录。找不到 `bash` 时让测试失败并说明 Bash 是该脚本的运行前提。

- [ ] **Step 2: 运行测试确认生成器尚不存在**

Run: `go test ./script -run TestGoMockgen_GeneratesMirroredMocksAndSkipsOutput -count=1`

Expected: FAIL，原因为 `script/go-mockgen.sh` 尚不存在。

- [ ] **Step 3: 实现 AST 扫描器和 Bash 入口**

`script/internal/mockscan/main.go` 接收 `-root` 与可重复的 `-exclude` 参数，用 `filepath.WalkDir` 扫描 `.go` 文件；跳过 `*_test.go`、`.git`、`vendor`、`.mocks` 和传入的输出树。每个文件用 `parser.ParseFile` 解析，仅当存在下列导出顶层声明时向标准输出写项目相对路径（一行一个文件）：

```go
type X interface { /* methods */ }
```

`script/go-mockgen.sh` 使用 `set -euo pipefail`，只接受必填的 `-o <output-root>`；从脚本位置解析项目根，创建输出根，并调用扫描器。对每个源文件执行：

```bash
go run go.uber.org/mock/mockgen@v0.6.0 \
  -source "$source_file" \
  -destination "$destination" \
  -package mocks
```

常规目标为：

```bash
destination="$output_root/$source_dir/mock_${source_basename}.go"
```

若相对路径含 `internal/`，取其前缀（例如 `cmd/app`）和 `internal/` 后缀（例如 `foo/contract.go`），目标改为：

```bash
destination="$project_root/cmd/app/.mocks/internal/foo/mock_contract.go"
```

根级 `internal/foo/contract.go` 使用 `$project_root/.mocks/internal/foo/mock_contract.go`。脚本不清空用户指定目录，只覆盖本次计算出的目标文件。

在 `server/mocks/README.md` 写明生成命令、常规镜像规则、`internal` 例外、禁止手改；在 `script/README.md` 的脚本表格加入同一条命令。文档明确 Git Bash 是 Windows 的执行环境，并给出 PowerShell 通过 Git Bash 可执行文件调用的示例。删除旧的扁平 `generate.go`，因为新入口不依赖 `go generate`。

- [ ] **Step 4: 运行脚本测试和格式化检查**

Run: `gofmt -w script/internal/mockscan/main.go script/go_mockgen_test.go && go test ./script -run TestGoMockgen_GeneratesMirroredMocksAndSkipsOutput -count=1`

Expected: PASS；测试清理后不存在 `server/.test-mocks`、`cmd/mockgenfixture` 或临时 `.mocks` 目录。

### Task 2: 将 Manager 收敛为业务模块并用生成 Mock 测试

**Files:**
- Create: `server/manager/health_check/interface.go`
- Create: `server/manager/health_check/manager.go`
- Create: `server/manager/health_check/local_probe.go`
- Create: `server/manager/health_check/README.md`
- Create: `server/manager/health_check/manager_test.go`
- Delete: `server/manager/health_check/health_check_impl.go`
- Delete: `server/manager/health_check/health_check_impl_test.go`

**Interfaces:**
- Produces: `HealthCheckManager.Check(context.Context) error`、`HealthProbe.Probe(context.Context) error`、`NewManager(HealthProbe) HealthCheckManager`、`NewLocalProbe() HealthProbe`。
- Consumes: `manager_mocks.NewMockHealthProbe(*gomock.Controller)`，其包路径为 `server/mocks/server/manager/health_check`。

- [ ] **Step 1: 将现有 Manager 测试改为嵌套 Mock 包的失败状态**

新测试继续覆盖成功与探针错误包装，且改用：

```go
manager_mocks "github.com/jasonlabz/generate-example-project/server/mocks/server/manager/health_check"

probe := manager_mocks.NewMockHealthProbe(ctrl)
probe.EXPECT().Probe(gomock.Any()).Return(probeErr)
```

同时保留 `TestLocalProbe_Probe_ReturnsNil`。此时嵌套 Mock 和模块契约尚不存在。

- [ ] **Step 2: 运行 Manager 测试确认缺少嵌套契约/Mock**

Run: `go test ./server/manager/health_check -count=1`

Expected: FAIL，错误指向不存在的嵌套 Mock 包或契约定义。

- [ ] **Step 3: 最小化迁移 Manager 契约和实现**

在 `interface.go` 放入：

```go
type HealthCheckManager interface { Check(context.Context) error }
type HealthProbe interface { Probe(context.Context) error }
```

`manager.go` 的私有结构体保存 `HealthProbe` 并将错误包装为 `fmt.Errorf("probe health: %w", err)`；`local_probe.go` 的私有空结构实现无 I/O 的成功探针。每个构造器返回本目录接口，保留 `var _` 编译期断言。

运行 `bash script/go-mockgen.sh -o server/mocks` 后，让测试导入生成的 `mock_interface.go`。README 说明 Manager 协调基础设施，直接下游仅是 Probe，测试只替换 Probe。旧根 `server/manager/health_check.go` 暂保留到 Wire 切换任务，防止中间状态破坏其它包。

- [ ] **Step 4: 验证 Manager 独立行为**

Run: `gofmt -w server/manager/health_check && go test ./server/manager/health_check -count=1`

Expected: PASS，成功时返回 nil，失败时可由 `errors.Is` 找到原始 Probe 错误。

### Task 3: 将 Service 与 Controller 依赖切换到模块内契约

**Files:**
- Create: `server/service/health_check/interface.go`
- Create: `server/service/health_check/types.go`
- Create: `server/service/health_check/service.go`
- Create: `server/service/health_check/README.md`
- Create: `server/service/health_check/service_test.go`
- Modify: `server/controller/health_check/controller.go`
- Modify: `server/controller/health_check/convertor.go`
- Modify: `server/controller/health_check/controller_test.go`
- Delete: `server/service/health_check/health_check_impl.go`
- Delete: `server/service/health_check/health_check_impl_test.go`
- Delete: `server/service/health_check/dto.go`
- Delete: `server/service/health_check/helper.go`

**Interfaces:**
- Produces: `HealthCheckService.Check(context.Context) (HealthCheckResult, error)`、`NewService(manager.HealthCheckManager) HealthCheckService`。
- Consumes: `manager/health_check.HealthCheckManager`；Controller consumes `service/health_check.HealthCheckService` and `HealthCheckResult`。

- [ ] **Step 1: 把 Service/Controller 测试的 import 改为目标模块 Mock 包**

Service 测试使用：

```go
manager_mocks "github.com/jasonlabz/generate-example-project/server/mocks/server/manager/health_check"
```

Controller 测试使用：

```go
service_mocks "github.com/jasonlabz/generate-example-project/server/mocks/server/service/health_check"
```

保持已有的四个行为：Service 的成功/错误包装、Controller 的既有成功 envelope 与内部错误 envelope。运行生成器前，Service Mock 缺失，测试必须失败。

- [ ] **Step 2: 运行目标层测试确认切换尚未完成**

Run: `go test ./server/service/health_check ./server/controller/health_check -count=1`

Expected: FAIL，错误指向缺少 Service 嵌套 Mock 或旧根 `service.HealthCheckResult` 的不匹配。

- [ ] **Step 3: 在 Service 业务目录定义契约、类型与实现，并更新 Controller**

`interface.go` 定义 `HealthCheckService`；`types.go` 定义：

```go
type HealthCheckResult struct { Status string }
```

`service.go` 保存 `health_check_manager.HealthCheckManager`，Manager 成功时返回 `HealthCheckResult{Status: "success"}`，失败时返回零值和 `fmt.Errorf("check health: %w", err)`。Controller 的字段、构造器、转换器均改为导入 `server/service/health_check`，不改变 HTTP operation、DTO 或 `humax.InternalServerError` 行为。

运行 `bash script/go-mockgen.sh -o server/mocks`，生成 Service Mock。Service README 说明它只表达应用用例并只依赖 Manager 接口；Controller README 的依赖图改为 Wire 组合根，并把旧 `go generate` 命令替换为新脚本命令。

- [ ] **Step 4: 验证 Controller 和 Service 的隔离测试**

Run: `gofmt -w server/service/health_check server/controller/health_check && go test ./server/service/health_check ./server/controller/health_check -count=1`

Expected: PASS；Controller 不导入 Manager/Probe，Service 不导入 Huma/Gin。

### Task 4: 引入模块化 Wire、瘦身 Router 并删除旧扁平层

**Files:**
- Create: `server/wire/health_check/wire.go`
- Create: `server/wire/README.md`
- Modify: `server/router/router.go`
- Modify: `server/router/router_test.go`
- Modify: `server/controller/health_check/README.md`
- Delete: `server/service/health_check.go`
- Delete: `server/manager/health_check.go`
- Delete: `server/mocks/mock_controller.go`
- Delete: `server/mocks/mock_manager.go`
- Delete: `server/mocks/mock_service.go`

**Interfaces:**
- Produces: `wire/health_check.NewHealthCheckController() controller.HealthCheckController`。
- Consumes: concrete `NewLocalProbe`, `NewManager`, `NewService` and `NewHealthCheckController` constructors.

- [ ] **Step 1: 增加 Router 静态路由移除和模块 Mock 的失败断言**

在 `router_test.go` 将 Controller Mock import 改为：

```go
controller_mocks "github.com/jasonlabz/generate-example-project/server/mocks/server/controller/health_check"
```

新增 `TestNewAPIRouter_DoesNotServeLegacyStaticFiles`，请求 `GET /server/`，断言 `http.StatusNotFound`。保留 `TestRegisterRootAPI_DelegatesToHealthCheckController`，但改用 `controller_mocks.NewMockHealthCheckController`。此时 Router 仍提供静态路由，回归测试必须失败。

- [ ] **Step 2: 运行 Router 测试确认静态路由仍存在**

Run: `go test ./server/router -run 'Test(NewAPIRouter_DoesNotServeLegacyStaticFiles|RegisterRootAPI_DelegatesToHealthCheckController)' -count=1`

Expected: FAIL，`GET /server/` 尚未返回 404，或 Controller Mock 包尚未刷新。

- [ ] **Step 3: 建立 Wire 组合根并清理 Router/旧层级**

`server/wire/health_check/wire.go` 只公开：

```go
func NewHealthCheckController() health_check_controller.HealthCheckController {
	probe := health_check_manager.NewLocalProbe()
	manager := health_check_manager.NewManager(probe)
	service := health_check_service.NewService(manager)
	return health_check_controller.NewHealthCheckController(service)
}
```

Router 删除直接的 Manager/Service import 和私有 `newHealthCheckController`，改为调用 Wire。删除 `staticRouter` 两行，不接触其它静态资源或构建配置。删除两个旧根契约和三个扁平生成 Mock；运行生成器重新写入路径化 Mock。Wire README 规定：每个业务模块一个 `server/wire/<module>/wire.go`，Wire 只能构建并连接具体实现，不定义业务接口、不承载 HTTP/领域转换。

- [ ] **Step 4: 验证 Router 组合与 HTTP 契约**

Run: `gofmt -w server/wire/health_check server/router && go test ./server/router -count=1`

Expected: PASS；`/health-check` 仍为原有四字段 envelope、无 `$schema` 和 `Link`，调试 Knife4go 文档仍为 OpenAPI 3.0.3，`/server/` 为 404。

### Task 5: 生成最终 Mock、完善演示文档并执行交付验证

**Files:**
- Modify: `server/controller/health_check/README.md`
- Modify: `server/service/health_check/README.md`
- Modify: `server/manager/health_check/README.md`
- Modify: `server/wire/README.md`
- Modify: `server/mocks/README.md`
- Modify: `script/README.md`
- Generated: `server/mocks/server/controller/health_check/mock_register.go`
- Generated: `server/mocks/server/service/health_check/mock_interface.go`
- Generated: `server/mocks/server/manager/health_check/mock_interface.go`

**Interfaces:**
- Consumes: 所有最终模块内 interface 声明。
- Produces: 文档化的可复制开发规则和与当前源码一致的 GoMock 文件。

- [ ] **Step 1: 刷新生成产物并检查旧 Mock 不再存在**

Run: `bash script/go-mockgen.sh -o server/mocks`

随后确认三份核心生成文件位于上述嵌套路径，且 `server/mocks/mock_controller.go`、`mock_service.go`、`mock_manager.go`、`generate.go` 均不存在。

- [ ] **Step 2: 对齐所有演示 README 的规则**

Controller README 说明 HTTP 适配职责与 Wire 装配；Service README 说明用例语义与 Manager Mock；Manager README 说明基础设施协调与 Probe Mock；Wire README 说明组合根；Mocks README 说明仅由脚本生成。每份 README 都给出 `bash script/go-mockgen.sh -o server/mocks` 和相应的 `go test` 范围，不重复实现细节。

- [ ] **Step 3: 运行格式、单元、竞态和构建验证**

Run:

```text
gofmt -l server/controller/health_check server/service/health_check server/manager/health_check server/wire script/internal/mockscan script/go_mockgen_test.go
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go build ./...
git diff --check
```

Expected: `gofmt -l` 无输出；全部 Go 命令退出码为 0；diff 检查无输出。

- [ ] **Step 4: 记录验证结果但不提交**

保留工作区变更，向用户报告生成命令、测试结果及未提交状态；不执行任何 Git 写入操作。

## Plan Self-Review

- 规格覆盖：Task 1 实现通用生成器与 internal 规则；Task 2/3 将 Manager、Service、Controller 迁为模块层；Task 4 完成 Wire、Router 装配与静态路由删除；Task 5 交付 README、最终生成物和完整验证。
- 一致性：所有 Mock import 均指向 `server/mocks/server/<layer>/health_check`，对应 `-o server/mocks` 的项目相对镜像；Wire 唯一公开构造器与 Router 调用名称一致。
- 范围：未引入新的业务能力、外部依赖、静态资源或构建脚本改动；明确遵守不提交约束。
