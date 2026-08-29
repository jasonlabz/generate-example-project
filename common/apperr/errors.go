// Package apperr defines the public error catalog for this generated service.
package apperr

import (
	"net/http"

	potatoErrors "github.com/jasonlabz/potato/errors"
)

const (
	// SystemCode must be allocated when this template becomes a real service.
	SystemCode = 10
)

// Spec defines one public error contract. It adds HTTP response policy to the
// immutable potato error definition; request-scoped causes stay on IError.
type Spec struct {
	HTTPStatus int

	base potatoErrors.IError
}

// Code returns the stable public code from the underlying potato definition.
func (s Spec) Code() int {
	return s.base.Code()
}

// Message returns the safe public message from the underlying potato definition.
func (s Spec) Message() string {
	return s.base.Message()
}

// WithErr returns the catalog error while retaining an internal cause.
func (s Spec) WithErr(cause error) potatoErrors.IError {
	return s.base.WithErr(cause)
}

// WithMessage customizes a safe message, such as a validation detail.
func (s Spec) WithMessage(message string) potatoErrors.IError {
	return s.base.WithMessage(message)
}

var (
	InvalidRequest        = define(100001001, http.StatusBadRequest, "请求参数不合法")
	Unauthenticated       = define(100002001, http.StatusUnauthorized, "登录凭证无效或已过期")
	Forbidden             = define(100003001, http.StatusForbidden, "无权执行此操作")
	NotFound              = define(100004001, http.StatusNotFound, "请求的资源不存在")
	Conflict              = define(100005001, http.StatusConflict, "资源状态已发生变化，请刷新后重试")
	BusinessRule          = define(100006001, http.StatusUnprocessableEntity, "当前操作不符合业务规则")
	DependencyUnavailable = define(100007001, http.StatusServiceUnavailable, "依赖服务暂不可用，请稍后重试")
	Internal              = define(100008001, http.StatusInternalServerError, "服务内部错误")
	RateLimited           = define(100009001, http.StatusTooManyRequests, "请求过于频繁，请稍后重试")

	byCode = index(InvalidRequest, Unauthenticated, Forbidden, NotFound, Conflict,
		BusinessRule, DependencyUnavailable, Internal, RateLimited)
)

func define(code, status int, message string) Spec {
	return Spec{
		HTTPStatus: status,
		base:       potatoErrors.New(code, message),
	}
}

func index(specs ...Spec) map[int]Spec {
	byCode := make(map[int]Spec, len(specs))
	for _, spec := range specs {
		byCode[spec.Code()] = spec
	}
	return byCode
}

// Lookup returns the public contract for a registered application error code.
func Lookup(code int) (Spec, bool) {
	spec, ok := byCode[code]
	return spec, ok
}

// ForHTTPStatus selects the public catalog entry used for framework errors.
func ForHTTPStatus(status int) Spec {
	switch status {
	case http.StatusUnauthorized:
		return Unauthenticated
	case http.StatusForbidden:
		return Forbidden
	case http.StatusNotFound:
		return NotFound
	case http.StatusConflict, http.StatusPreconditionFailed:
		return Conflict
	case http.StatusUnprocessableEntity:
		return BusinessRule
	case http.StatusTooManyRequests:
		return RateLimited
	default:
		if status >= http.StatusInternalServerError {
			return Internal
		}
		return InvalidRequest
	}
}
