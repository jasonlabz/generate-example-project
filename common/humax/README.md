# humax — huma 响应、分页与文件流封装

为 huma 处理器提供统一响应信封、分页输出、带状态码的错误封装和文件流输出。

## 类型

| 类型 | 用途 | 对应 swag |
|------|------|-----------|
| `Output[T]` | 成功响应体（泛型信封） | `@Success` 返回结构 |
| `PaginationOutput[T]` | 带分页元数据的成功响应 | 分页 `@Success` 返回结构 |
| `Error` | 统一错误响应（error + StatusError） | `@Failure` + 错误结构 |
| `Wrap` / `WrapList` / `WrapPage` | 将 Controller handler 适配为统一信封 | 列表 / 单值 / 分页接口 |
| `Success[T]` | 构造成功响应 | — |
| `BusinessError` | 协议层面的预期失败（参数合法但组合不支持） | 业务/参数错误 |
| `FromError` / `MapError` | 把 service 错误映射为统一错误响应 | `@Failure` |
| `Result` / `PaginationResult` | 成功结果与错误的统一返回入口 | — |
| `File` / `SimpleFile` | Huma 文件流响应 | 二进制文件响应 |
| `InternalServerError` | 500 统一错误 | `@Failure 500` |

## 使用

```go
// 成功
return humax.Success(consts.APIVersionV1, data), nil

// 未知内部错误（HTTP 500，message 不暴露 cause）
return nil, humax.InternalServerError(consts.APIVersionV1, err)

// 业务失败（推荐走 apperr 目录，HTTP 状态与 code 都由目录决定）
return nil, apperr.NotFound.WithMessage(fmt.Sprintf("用户 %d 不存在", id))

// 协议层面的业务失败：参数合法但组合不支持（HTTP 422 + code 100006001）
return nil, humax.BusinessError(consts.APIVersionV1, apperr.BusinessRule.Code(), "暂不支持 xlsx 导出")

// 分页
return humax.PaginationSuccess(consts.APIVersionV1, rows, pagination), nil

// 文件流
return humax.SimpleFile(consts.APIVersionV1, filePath, fileName)
```

Controller 注册列表接口用 `humax.Wrap`（handler 返回 `[]T`，`data` 直接透传），
单值接口用 `humax.WrapList`（handler 返回 `T`，`data` 自动包一层 `[T]`），
分页接口返回 `([]T, *humax.Pagination, error)` 并用 `humax.WrapPage`。
`Wrap` / `WrapList` 都会把 nil 归一化为对应类型的零值（nil 切片→`[]`、nil map→`{}`、nil 指针→零值结构体）。
列表查询参数统一使用 `page default:"1" minimum:"1"` 与
`page_size default:"200" minimum:"1" maximum:"200"`；
筛选型 query 参数保持可选，业务必须依赖的参数才使用 `required:"true"`。

字节下载接口用 `humax.BinaryResponse` 在 `Operation.Responses` 里声明具体 media type 与 binary schema，
handler 返回 `(*huma.StreamResponse, error)`。

> `FileResult` / `FileResultWithError` / `SimpleFileDownload` 是 `File` / `FileWithError` / `SimpleFile`
> 的同义别名，仅为兼容既有调用保留；新代码统一使用后者。

## 错误契约

HTTP 状态与业务 code 是双轨：状态让调用方做通用处理，code 让调用方精确定位。

```text
成功            HTTP 200  code = 0
参数校验失败     HTTP 422  code = 100001001   huma 自动映射
业务失败         HTTP 404/409/422/403/...      apperr 目录中登记的 code
未知内部错误     HTTP 500  code = 100008001   对外恒定文案
```

- 成功响应统一 `Envelope`（版本 + code/message/data）；错误响应复用同一结构，不退回 RFC7807。
- 业务失败优先用 `apperr` 目录：`humax.FromError` 用 `errors.As` 在**整条错误链**里定位
  `potato/errors.IError`，因此 `fmt.Errorf("...: %w", apperrErr)` 包装后 code 与 HTTP 语义依然保留
  （实测包两层也不丢）。真正会丢的是 `%v`——它截断错误链，链断后就只能降级成 500。
- `humax.BusinessError(version, code, message)` 用于协议层面的失败；未登记的 code 回退为
  `InvalidRequest` 的 HTTP 语义，但保留传入的 code 与 message。
- Huma 参数校验由 `ConfigureHumaErrorFactory` 转为 HTTP 422 + code 100001001。
  huma 用 422 表示参数不合法，`apperr.ForHTTPStatus` 把它归 `InvalidRequest` 但保留 422 状态；
  否则会和业务代码主动表达的 `BusinessRule`（同为 422）撞码。
- 只有未知内部错误返回 HTTP 500，响应固定为 `服务内部错误`，不泄漏内部 cause。

### message 与 err_trace 的分工

`message` 是概括性文案（给调用方展示），`err_trace` 是详细报错链（给开发者定位）。

`ConfigureErrorDetails(true)`（由 Router 按 `gin.IsDebugging()` 驱动）打开后，
**任何错误都会带上 `err_trace`**，5xx 与 4xx 一视同仁：

| 错误构造 | HTTP | err_trace |
|----------|------|-----------|
| `fmt.Errorf("query user: %w", dbErr)` | 500 | `"query user: connection refused"` |
| `apperr.NotFound.WithErr(dbErr)` | 404 | `"100004001 -> 请求的资源不存在\n inner error: db: timeout"` |
| `apperr.NotFound.WithMessage("用户 9 不存在")` | 404 | `"100004001 -> 用户 9 不存在"`（回退到目录链） |
| `humax.BusinessError(v1, 100006001, "暂不支持 xlsx")` | 422 | `"100006001 -> 暂不支持 xlsx"` |
| Huma 参数校验 | 422 | `"100001001 -> query.page_size: 必须小于等于200"` |

实现见 `detailTrace`：能拿到 `cause` 就输出它的完整 `Error()`；拿不到（如纯 `WithMessage`
构造的业务错误）就回退到目录错误本身，保证调试时 `err_trace` 始终有内容。

生产环境关闭该开关时，`err_trace` 字段完全省略（`omitempty`），内部原因不出现在响应中。

> 注意：关闭开关后内部原因**也不会进入服务端日志**——`HumaRequestMiddleware` 的
> `error_message` 取自 `gin.Errors`，而 humagin 适配器不会调用 `gin.Error()`，
> 该字段恒为空。需要生产可排查时，应自行在错误出口记录日志（见根 README 待办）。

## 原理

huma 约定：handler 返回的 `Error` 实现 `huma.StatusError`（`GetStatus() int`）。
本包由此把可预期失败映射到 `apperr` 目录登记的 HTTP 状态，未知故障固定为 500。
