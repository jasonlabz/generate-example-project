// Package humax 为 huma 处理器提供统一响应信封与错误封装。
//
// 用途：
//   - Output[T]：成功响应体（对应 swag 的 @Success 返回结构）；
//   - Error：带状态码的错误响应（实现 huma.StatusError 接口，
//     对应 swag 的 @Failure + response struct）。
//
// huma 的错误约定：handler 返回 error 时，若 error 实现了
// huma.StatusError 接口（GetStatus() int），huma 使用其状态码；
// 否则默认 200/500。因此错误必须显式携带状态码。
package humax

import (
	"errors"
	"net/http"

	"github.com/jasonlabz/generate-example-project/common/response"
)

// Output 是 huma 处理器的成功响应体（泛型）。
// 用法：handler 返回 &humax.Output[T]{Body: response.New(version, data)}。
type Output[T any] struct {
	Body *response.Envelope[T]
}

// Success 构造成功响应（统一信封 response.Envelope）。
func Success[T any](version string, data T) *Output[T] {
	return &Output[T]{
		Body: response.New(version, data),
	}
}

// Error 是带 HTTP 状态码的统一错误响应。
// 同时实现 error 与 huma.StatusError 接口，可直接作为 handler 返回值。
type Error struct {
	*response.Envelope[[]any]
	status int
	cause  error
}

// InternalServerError 把意外错误转换为 500 统一错误响应。
// 对应 swag 的 @Failure 500 场景；业务错误可仿照此函数增加
// BadRequestError/NotFoundError 等带对应状态码的构造。
func InternalServerError(version string, cause error) *Error {
	if cause == nil {
		cause = errors.New(http.StatusText(http.StatusInternalServerError))
	}

	return &Error{
		Envelope: response.NewError(version, []any{}, 0, cause.Error(), cause.Error()),
		status:   http.StatusInternalServerError,
		cause:    cause,
	}
}

// Error 实现 error 接口，返回底层错误文本。
func (e *Error) Error() string {
	return e.cause.Error()
}

// GetStatus 实现 huma.StatusError 接口，huma 据此写入 HTTP 状态码。
func (e *Error) GetStatus() int {
	return e.status
}
