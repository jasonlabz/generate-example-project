# 后端分层规范

> Gin 项目模板的目录职责与依赖规则，写代码前先对照此文件。

## 目录职责

| 目录 | 职责 |
|------|------|
| `cmd/<name>/` | 二进制入口，只做 bootstrap + 启动，不写业务 |
| `internal/bootstrap/` | 配置加载、资源初始化编排（含 dao 装载） |
| `internal/server/` | HTTP/gRPC/pprof 启动与优雅退出 |
| `internal/router/` | 路由分组与中间件装配 |
| `internal/controller/` | HTTP 入参校验与响应组装 |
| `internal/rpc/` | gRPC 服务实现（proto 接口的薄适配层） |
| `internal/service/` | 业务逻辑；`{module}.go` 接口 + `{module}/impl.go` 实现 + `{module}/dto.go` |
| `internal/dal/db/` | 数据访问层，gentol 生成 dao/model 的落点 |
| `internal/client/` | 外部服务调用，配合 `conf/servicer/*.yaml` |
| `internal/middleware/` | 项目自有 gin 中间件 |
| `internal/resource/` | 全局客户端单例（logger/RMQ/Redis/ES） |
| `pkg/` | 可复用库（ginx/consts/helper），不依赖 internal |
| `api/proto/` | proto 定义与生成物 |

## 依赖规则（强制）

1. controller/rpc 只做入参校验和响应组装，不写业务逻辑、不碰 dao
2. service 不 import gin（入参用 dto，保持可脱离 HTTP 测试）
3. dao 不 import service（防循环依赖）
4. pkg 不 import internal（保证 pkg 可被抽走复用）

## 模式约定

- service：接口定义在 `internal/service/{module}.go`，实现用 `GetService()` + `sync.Once` 单例（`once.Do` 内初始化，不写 `if svc != nil` 提前返回）
- dao：`NewXxxDao(db)` 构造 + `GetXxxDao()` 取默认装载连接；单测传内存 SQLite（见 `internal/dal/db/dao/user_test.go`）
- 新增 HTTP 接口：controller 方法 + `internal/router` 注册 + service 接口/实现 + swagger 注释
- 新增 gRPC 接口：`api/proto` 定义 → `bash script/proto.sh` 生成 → `internal/rpc` 实现 → `rpc.Register` 挂载
- 响应统一走 `pkg/ginx`（`JsonResult` / `PaginationResult` / `FileResult`）
