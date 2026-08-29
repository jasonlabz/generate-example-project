# apperr

`apperr` 是生成服务的公开错误目录。模板当前采用 `10MMTTDDD`；项目首次落地时应先分配系统号，再统一调整目录中的业务码。

## Spec 与 potatoErrors.IError

`potatoErrors.IError` 是一次错误发生时的载体：它保存错误码、可展示文案和可选的内部根因，`WithErr` 与 `WithMessage` 会返回新的错误实例。

`Spec` 是静态的公开响应契约：它引用一个已注册的 `IError` 定义，并补充 HTTP 状态。Potato 保持与 HTTP 无关，因此不能把状态码放进 `IError`。`Spec.Code()` 和 `Spec.Message()` 直接读取它引用的 Potato 定义，避免同一业务码或文案重复保存。

请求路径的流程是：业务层返回 `InvalidRequest.WithMessage(...)` 或 `Internal.WithErr(...)`；`humax` 用该 `IError` 的 code 调用 `Lookup` 解析对应 `Spec`；最后按 `Spec.HTTPStatus` 和安全文案组装响应。未知或非本目录的 `IError` 会统一映射为安全的内部错误，不会直接暴露给调用方。

新增公开错误时只需定义一次：

```go
var DataSourceUnavailable = define(100307001, http.StatusServiceUnavailable, "数据源暂不可用，请稍后重试")
```

再把它加入 `index(...)`。不要直接复用 Potato 的通用或 legacy 错误码。
