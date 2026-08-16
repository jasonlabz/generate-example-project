# router 路由层

路由组装：huma 组层级、gin 中间件、knife4go 文档注册。

## 路由架构

```text
gin.Engine（底层，承载 gin 中间件与非 huma 资产）
└── rooterAPI（huma API，根级；health-check 等根路由注册在此）
    └── serverAPI（huma 组，前缀 /<服务名>）
        └── apiGroup（huma 组，前缀 /api）
            └── v1Group（huma 组，前缀 /v1）→ /<服务名>/api/v1/**
```

- `huma.NewGroup` 的组前缀同时写入路由与 OpenAPI 文档 paths。
- 根级路由（如 health-check）注册在 `rooterAPI`，其文档 paths 无前缀；
  业务路由注册在 `serverAPI` 组下，文档 paths 带服务名前缀。

## 文档注册（knife4go）

- 仅调试模式（`gin.IsDebugging()`）注册；
- 文档用 `serverAPI.OpenAPI().Downgrade()` 生成，**必须放在所有路由注册之后**（文档才完整）；
- 文档 paths 带前缀，knife4j 前端检测到后不再拼接 doc.html 位置前缀；
- 文档地址：`http(s)://ip:port/<服务名>/doc.html`。

## 中间件

- gin 中间件（`server/middleware/`、potato）：`rootMiddleware` 中 Engine 级注册，作用于全部路由；
- huma 中间件（`huma.Middlewares`）：按组挂载（`huma.Group.UseMiddleware`），
  `registerRootAPI`/`registerBaseAPI`/`registerV1GroupAPI` 的 `middleware` 参数预留入口。

## 新增模块注册

业务模块控制器在 `registerV1GroupAPI` 中注册：

```go
user.NewController().Register(api)   // 挂到 /<服务名>/api/v1/**
```

完整说明见[根目录 README](../../README.md)。
