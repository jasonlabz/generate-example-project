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
| `BusinessError` / `FromError` | 预期失败（HTTP 200 + 非零 code） | 业务/参数错误 |
| `Result` / `PaginationResult` | 成功结果与错误的统一返回入口 | — |
| `File` / `SimpleFile` | Huma 文件流响应 | 二进制文件响应 |
| `InternalServerError` | 500 统一错误 | `@Failure 500` |

## 使用

```go
// 成功
return humax.Success(consts.APIVersionV1, data), nil

// 未知内部错误（HTTP 500，message 不暴露 cause）
return nil, humax.InternalServerError(consts.APIVersionV1, err)

// 预期业务错误（HTTP 200，code 必须非零）
return nil, humax.BusinessError(consts.APIVersionV1, 1001, "资源不存在")

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
`page_size default:"200" minimum:"1" maximum:"500"`；
筛选型 query 参数保持可选，业务必须依赖的参数才使用 `required:"true"`。

字节下载接口用 `humax.BinaryResponse` 在 `Operation.Responses` 里声明具体 media type 与 binary schema。

`FileResult`、`FileResultWithError`、`SimpleFileDownload` 提供显式的文件结果命名；Huma handler 应返回 `(*huma.StreamResponse, error)`。

## 错误契约

- 成功与可预期失败均返回 HTTP 200；失败通过非零 `code` 区分。
- `Operation.DefaultStatus` 固定为 200，`Operation.Errors` 仅声明 500。
- Huma 参数校验由 `ConfigureHumaErrorFactory` 转为 HTTP 200、`code=1` 的 Envelope。
- `potato/errors.IError` 会保留其业务 code 和公开 message；其他未知错误才映射为 500。
- 只有未知内部错误返回 HTTP 500，响应固定为 `Internal Server Error`，不暴露 `err_trace` 或 cause。

## 原理

huma 约定：handler 返回的 `Error` 实现 `huma.StatusError`（`GetStatus() int`）。
本模板将可预期错误固定为 200，未知内部错误固定为 500。
