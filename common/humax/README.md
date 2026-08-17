# humax — huma 响应、分页与文件流封装

为 huma 处理器提供统一响应信封、分页输出、带状态码的错误封装和文件流输出。

## 类型

| 类型 | 用途 | 对应 swag |
|------|------|-----------|
| `Output[T]` | 成功响应体（泛型信封） | `@Success` 返回结构 |
| `PaginationOutput[T]` | 带分页元数据的成功响应 | 分页 `@Success` 返回结构 |
| `Error` | 带状态码的错误响应（error + StatusError） | `@Failure` + 错误结构 |
| `Success[T]` | 构造成功响应 | — |
| `Result` / `PaginationResult` | 成功结果与错误的统一返回入口 | — |
| `File` / `SimpleFile` | Huma 文件流响应 | 二进制文件响应 |
| `InternalServerError` | 500 统一错误 | `@Failure 500` |

## 使用

```go
// 成功
return humax.Success(consts.APIVersionV1, data), nil

// 错误（必须携带状态码）
return nil, humax.InternalServerError(consts.APIVersionV1, err)

// 分页
return humax.PaginationSuccess(consts.APIVersionV1, rows, pagination), nil

// 文件流
return humax.SimpleFile(consts.APIVersionV1, filePath, fileName)
```

`FileResult`、`FileResultWithError`、`SimpleFileDownload` 提供显式的文件结果命名；Huma handler 应返回 `(*huma.StreamResponse, error)`。

## 扩展新状态码

仿照 `InternalServerError` 增加构造函数：

```go
// NotFoundError 返回 404 统一错误响应。
func NotFoundError(version string, cause error) *Error {
	if cause == nil {
		cause = errors.New(http.StatusText(http.StatusNotFound))
	}
	return &Error{
		Envelope: NewError(version, []any{}, 0, cause.Error(), cause.Error()),
		status:   http.StatusNotFound,
		cause:    cause,
	}
}
```

## 原理

huma 约定：handler 返回 error 时，若 error 实现 `huma.StatusError` 接口
（`GetStatus() int`），huma 使用其状态码；否则默认 200/500。
`Error.GetStatus()` 即为此接口实现。
