# humax — huma 响应封装

为 huma 处理器提供统一响应信封与带状态码的错误封装。

## 类型

| 类型 | 用途 | 对应 swag |
|------|------|-----------|
| `Output[T]` | 成功响应体（泛型信封） | `@Success` 返回结构 |
| `Error` | 带状态码的错误响应（error + StatusError） | `@Failure` + 错误结构 |
| `Success[T]` | 构造成功响应 | — |
| `InternalServerError` | 500 统一错误 | `@Failure 500` |

## 使用

```go
// 成功
return humax.Success(apiVersion, data), nil

// 错误（必须携带状态码）
return nil, humax.InternalServerError(apiVersion, err)
```

## 扩展新状态码

仿照 `InternalServerError` 增加构造函数：

```go
// NotFoundError 返回 404 统一错误响应。
func NotFoundError(version string, cause error) *Error {
	if cause == nil {
		cause = errors.New(http.StatusText(http.StatusNotFound))
	}
	return &Error{
		Envelope: response.NewError(version, []any{}, 0, cause.Error(), cause.Error()),
		status:   http.StatusNotFound,
		cause:    cause,
	}
}
```

## 原理

huma 约定：handler 返回 error 时，若 error 实现 `huma.StatusError` 接口
（`GetStatus() int`），huma 使用其状态码；否则默认 200/500。
`Error.GetStatus()` 即为此接口实现。
