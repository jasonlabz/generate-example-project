# server 目录约定

服务端分层代码，遵循「单向调用链」：`HTTP → controller → service → manager → DAO/外部系统`。

## 目录职责

```text
server/
  router/                 # 根路由与中间件分层装配（huma 组层级 + knife4go 文档）
  controller/             # HTTP Controller，只处理入参、调用 service、返回响应
  wire/                   # 组合根；按模块组装 Controller/Service/Manager/DAO
  service/
    {module}/
      interface.go        # Service 接口
      service.go          # Service 实现
      types.go            # Service 结果类型
  manager/{module}/       # 业务编排与 DAO/Probe 接口依赖
  middleware/             # gin 中间件（上下文、日志）
dal/db/dao/               # DAO 接口与数据库适配实现（gentol 生成）
```

## 分层职责（强制）

| 层 | 职责 | 禁止 |
|----|------|------|
| controller | 路由注册、参数/响应模型（DTO）、模型转换 | 业务逻辑、直连 DB |
| service | 业务用例编排（调用 manager 组合能力） | 直接访问 DAO/外部系统 |
| manager | 单一技术能力（DB、Redis、外部 API） | 业务判断 |
| wire | 依赖装配 | 逻辑 |

## 新增模块

以 `user` 模块为例，按 `health_check` 的结构复制：

```text
server/controller/user/       # register.go / controller.go / types.go / convertor.go
server/service/user/          # interface.go / service.go / types.go
server/manager/user/          # interface.go / manager.go / 具体依赖实现
server/wire/user/wire.go      # 对象图装配
```

Controller 不写业务逻辑，Service 不依赖 Gin。新增模块在 `server/wire/user/wire.go` 中装配对象图，并在 `server/router/router.go` 的 `registerRootAPI` 或 `registerV1GroupAPI` 中注册路由。中间件在 Engine 作用域统一注入；认证、授权等受保护逻辑放在受保护路由组内，不要放在公开根路由。

## Huma 约定

- 路由用 `huma.Register(api, huma.Operation{...}, handler)` 注册，`Operation` 字段对应 swag 注解（对照表见根 README）。
- 请求/响应模型：结构体 + huma tag（`path`/`query`/`header`/`body`）+ 统一响应信封 `response.Envelope`。
- 错误必须携带状态码：返回 `common/humax.Error`（实现 `huma.StatusError`）。
- 控制器规范见 [controller/README.md](controller/README.md)。

## 相关文档

- [根目录 README](../README.md)：Huma 编写规范模板、swag→huma 对照、注意事项
- [controller/README.md](controller/README.md) / [service/README.md](service/README.md) / [manager/README.md](manager/README.md)
