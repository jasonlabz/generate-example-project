# controller 控制器层

HTTP 控制器负责 Huma 路由（Operation）、请求/响应模型（DTO）和 DTO 与业务模型的转换，**不包含业务逻辑**。

## 目录结构（每个业务域一个目录）

```text
server/controller/<module>/
├── register.go                    包说明 + Controller.Register + Huma operation
├── <module>_controller.go         具体 Controller、构造函数和 huma handler
├── types.go                       DTO：请求/响应模型（huma tag + humax.Envelope）
└── convertor.go                   业务模型 ↔ DTO 转换函数
```

## 代码风格

- 默认使用具体 `Controller` 和 `NewController(service.Service) *Controller`，不为了调用 `Register` 额外定义接口或私有实现。
- `Register(api)` 可以在一次调用中注册本业务域的多个 path；每个 handler 调用 Service 的对应方法。
- 当调用方确实需要替换多种 Controller 实现时，才抽取最小接口。
- 普通 handler 签名：`func(ctx context.Context, in *In) (*Out, error)`，注册时使用 `humax.Wrap`。
- 分页 handler 签名：`func(ctx context.Context, in *In) ([]Out, *humax.Pagination, error)`，注册时使用 `humax.WrapPage`。
- 文件流 handler 直接返回 `*huma.StreamResponse`（用 `humax.File` 构造），不经过 `Wrap`。
- Operation 必须声明 `DefaultStatus: http.StatusOK`；`Errors` 声明该接口实际会返回的状态码，
  例如 `[]int{http.StatusNotFound, http.StatusInternalServerError}`，保证文档与行为一致。
- query 筛选参数不传时不筛选；只有业务不能执行的参数添加 `required:"true"`。分页统一 `page=1`、`page_size=200`，且 `page_size` 最大为 200。
- handler 不转换错误：service 抛出的 `apperr` 错误原样上抛，由 `humax.Wrap` 统一映射，
  错误码才不会在传递中丢失。协议层面的业务失败用 `humax.BusinessError`。
- DTO 转换集中在 `convertor.go`，handler 保持简洁。

完整模板与 swag→huma 对照见[根目录 README](../../README.md)。
