# humax — huma 响应、分页与文件流封装

为 huma 处理器提供统一响应信封、分页输出、带状态码的错误封装和文件流输出。

## 类型

| 类型 | 用途 | 对应 swag |
|------|------|-----------|
| `Output[T]` | 成功响应体（泛型信封） | `@Success` 返回结构 |
| `PaginationOutput[T]` | 带分页元数据的成功响应 | 分页 `@Success` 返回结构 |
| `Error` | 统一错误响应（error + StatusError） | `@Failure` + 错误结构 |
| `Success[T]` | 构造成功响应 | — |
| `BusinessError` | 预期失败（HTTP 200 + 非零 code） | 业务/参数错误 |
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

`FileResult`、`FileResultWithError`、`SimpleFileDownload` 提供显式的文件结果命名；Huma handler 应返回 `(*huma.StreamResponse, error)`。

## 错误契约

- 成功与可预期失败均返回 HTTP 200；失败通过非零 `code` 区分。
- Huma 参数校验由 `ConfigureHumaErrorFactory` 转为 HTTP 200、`code=1` 的 Envelope。
- 只有未知内部错误返回 HTTP 500，响应固定为 `Internal Server Error`，不暴露 `err_trace` 或 cause。

## 原理

huma 约定：handler 返回的 `Error` 实现 `huma.StatusError`（`GetStatus() int`）。
本模板将可预期错误固定为 200，未知内部错误固定为 500。
