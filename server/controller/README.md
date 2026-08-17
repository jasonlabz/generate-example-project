# controller 控制器层

HTTP 控制器：定义 huma 路由（Operation）、请求/响应模型（DTO）、并做 DTO 与业务模型的转换。**不包含业务逻辑**。

## 目录结构（每个业务模块一个目录）

```text
server/controller/<module>/
├── register.go    包说明 + 路由注册（huma.Register + huma.Operation）
├── controller.go  控制器实现（构造函数注入 service 依赖 + huma handler）
├── types.go       DTO：请求/响应模型（huma tag + humax.Envelope）
└── convertor.go   业务模型 ↔ DTO 转换函数
```

## 代码风格

- 控制器形态：`接口 + 私有实现 + 构造函数`（`NewXxxController(service) XxxController`），便于 mock 与测试。
- 编译期断言：`var _ XxxController = (*controller)(nil)`。
- 路由注册：

```go
huma.Register(api, huma.Operation{
	OperationID: "user-get",
	Method:      http.MethodGet,
	Path:        "/users/{id}",
	Summary:     "获取用户",
	Description: "按 ID 获取用户信息。",
	Tags:        []string{"用户"},
	DefaultStatus: http.StatusOK,
	Errors: []int{http.StatusNotFound, http.StatusInternalServerError},
}, c.handleGet)
```

- handler 签名：`func(ctx context.Context, in *In) (*Out, error)`。
- 错误处理：业务错误返回 `humax` 的带状态码错误（`InternalServerError` 等，需新状态码时仿照增加）。
- 转换逻辑集中 `convertor.go`，handle 保持简洁。

完整模板与 swag→huma 对照见[根目录 README](../../README.md)。
