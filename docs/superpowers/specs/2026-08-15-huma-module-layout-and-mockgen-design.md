# Huma 分层模块与 Mock 生成设计

**日期：** 2026-08-15  
**状态：** 已确认，待实现  
**范围：** `generate-example-project` 的 Health Check 演示模块；不提交代码。

## 目标

将 Health Check 做成可复制的 Huma 分层示例：业务按模块组织、依赖由独立的 Wire 组合根装配、各层可通过 GoMock 替身独立测试。删除路由中已不再需要的 `staticRouter`，并提供可复用的 Mock 生成脚本。

## 不在范围内

- 不修改 Makefile、Dockerfile、镜像构建脚本或 `webroot` 资源；本次只删除 Gin 路由中的静态服务。
- 不保留旧的扁平层级文件与扁平 `server/mocks` 生成物。
- 不提交或推送任何变更。

## 模块边界与目录

```text
server/
  controller/health_check/
    README.md       # HTTP 边界、DTO、注册和测试规则
    register.go     # HealthCheckController 接口及 Huma 路由注册
    controller.go   # 请求编排，依赖服务接口
    types.go        # HTTP 输入/输出 DTO
    convertor.go    # 领域结果到 HTTP DTO 的转换
  service/health_check/
    interface.go    # HealthCheckService 接口
    types.go        # 服务层结果类型
    service.go      # 服务实现，依赖管理器接口
    README.md       # 服务层职责与依赖规则
  manager/health_check/
    interface.go    # HealthCheckManager、HealthProbe 接口
    manager.go      # 管理器实现，依赖探针接口
    local_probe.go  # 本地探针实现
    README.md       # 管理器/基础设施职责与依赖规则
  wire/health_check/
    wire.go         # 该业务模块唯一的对象图装配点
  wire/README.md    # 组合根的边界与新增模块规则
  mocks/
    ...             # 由脚本生成；按源码路径镜像
    README.md       # 生成命令、路径规则与禁止手改约束
script/
  go-mockgen.sh     # 项目级 Mock 生成入口
```

Controller、Service、Manager 都以 `health_check` 业务目录为边界。Controller 因为承担 HTTP 适配而保留 `register.go`、DTO 和转换器；Service、Manager 只保留其职责所需的接口、类型和实现文件，不人为制造同名空文件。

## 依赖与装配

```text
Gin Router -> Wire(health_check) -> Controller -> Service -> Manager -> Probe
```

- Router 只注册路由并向 `wire/health_check` 取得 Controller，不创建 Probe、Manager 或 Service，避免成为服务定位器。
- `wire/health_check` 是组合根，只引用具体构造器；不定义业务接口、不承载转换逻辑。
- Controller 持有 `service/health_check.HealthCheckService`；Service 持有 `manager/health_check.HealthCheckManager`；Manager 持有 `manager/health_check.HealthProbe`。
- 接口放在使用它的业务模块内，契约、实现与类型邻近，既可依赖倒置又不会形成共享的全局 interface 包。
- 每个新增业务模块按同一规则增设 `server/wire/<module>/wire.go`。

## 静态路由

仅从 `server/router/router.go` 删除 `staticRouter` 及 `/server/*` 静态路由。打包资源与构建脚本保持不动，这是明确的范围边界；因此相关脚本中的旧访问提示本轮不作修订。

## Mock 生成规则

`script/go-mockgen.sh -o <输出根目录>` 是唯一入口，默认扫描项目内所有非测试 Go 源文件，并只为可跨包引用的导出 interface 生成外部 Mock：

在 Windows 上从 Git Bash 执行该入口；系统的 WSL 启动器若未配置 Linux 发行版，不能替代 Git Bash。

1. 跳过 `*_test.go`、Git/Vendor 目录、`-o` 指向的输出树和所有 `.mocks/` 目录，避免递归生成。
2. 用 Go AST 识别源文件的导出 interface；私有 interface 可能使用同包私有类型，不能由外部 `mocks` 包合法表达，因此不生成 Mock。
3. 对每个含接口的源文件以固定版本的 `go.uber.org/mock/mockgen` source mode 生成，输出包名为 `mocks`。
4. 常规源码按项目相对路径镜像：`server/service/health_check/interface.go` 对应 `<输出根>/server/service/health_check/mock_interface.go`。
5. 对处在 `internal/` 下的源码，为满足 Go internal 导入可见性，输出改放在该 `internal` 父目录下的 `.mocks/`：例如 `cmd/app/internal/foo/interface.go` 对应 `cmd/app/.mocks/internal/foo/mock_interface.go`。这是唯一的路径镜像例外。
6. `server/mocks` 的旧扁平生成文件和旧 `go:generate` 声明一并移除；测试改为导入各自模块的生成包。

## 测试与验收

- Controller、Service、Manager 各有成功和错误/边界样例，并通过对应模块 Mock 隔离下游依赖。
- Router 保留 HTTP 契约、Huma 文档开关和装配委托测试，新增 `/server/` 不再由路由提供静态内容的回归测试。
- Mock 脚本覆盖常规镜像、输出目录排除和 `internal` 例外三种路径规则。
- 运行生成脚本后，`go test -count=1 ./...`、`go vet ./...`、`go build ./...`、目标代码格式检查和 `git diff --check` 都必须通过。

## 开发规则（供后续模块复制）

- 向内依赖：HTTP DTO 不进入 Service；Huma/Gin 不进入 Service、Manager。
- 构造器显式接收接口依赖；生产实现只在 Wire 中连接，测试直接注入 Mock。
- 一个业务模块的公开契约、类型和实现必须在同一业务目录中；跨模块只依赖上游所需的最小接口。
- README 记录职责、依赖方向、构造方式和测试替身位置；不复制实现细节。
