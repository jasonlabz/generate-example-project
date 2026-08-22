package humax

import (
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Error is a uniform error response that implements huma.StatusError.
// Its cause remains available to server-side callers but is never serialized.
type Error struct {
	*Envelope[[]any]
	status int
	cause  error
}

// InternalServerError converts an unexpected error into a safe 500 response.
func InternalServerError(version string, cause error) *Error {
	if cause == nil {
		cause = errors.New(http.StatusText(http.StatusInternalServerError))
	}
	return &Error{
		Envelope: NewError(version, []any{}, 0, http.StatusText(http.StatusInternalServerError), ""),
		status:   http.StatusInternalServerError,
		cause:    cause,
	}
}

// BusinessError creates a public error using the shared envelope.
// Expected failures always use HTTP 200 and a non-zero business code.
func BusinessError(version string, code int, message string) *Error {
	if message == "" {
		message = http.StatusText(http.StatusOK)
	}
	return &Error{
		Envelope: NewError(version, []any{}, code, message, ""),
		status:   http.StatusOK,
		cause:    errors.New(message),
	}
}

// MapError preserves shared status errors and converts unexpected errors to the shared envelope.
func MapError(version string, err error) error {
	if err == nil {
		return nil
	}

	var sharedError *Error
	if errors.As(err, &sharedError) {
		return sharedError
	}
	return InternalServerError(version, err)
}

// ConfigureHumaErrorFactory makes Huma input errors use the shared envelope.
// Call it once during router setup before registering operations.
func ConfigureHumaErrorFactory(version string) {
	newError := func(status int, message string, details ...error) huma.StatusError {
		if status >= http.StatusInternalServerError {
			return InternalServerError(version, errors.New(message))
		}
		return BusinessError(version, 1, validationMessage(message, details))
	}
	huma.NewError = newError
	huma.NewErrorWithContext = func(_ huma.Context, status int, message string, details ...error) huma.StatusError {
		return newError(status, message, details...)
	}
}

// Error implements error.
func (e *Error) Error() string {
	if e == nil || e.cause == nil {
		return ""
	}
	return e.cause.Error()
}

// GetStatus implements huma.StatusError.
func (e *Error) GetStatus() int {
	return e.status
}

// ContentType keeps error responses on the JSON media type used by the shared envelope.
func (*Error) ContentType(contentType string) string {
	return strings.Replace(contentType, "problem+", "", 1)
}

func validationMessage(message string, details []error) string {
	for _, detail := range details {
		if detail == nil {
			continue
		}
		if errorDetail, ok := detail.(huma.ErrorDetailer); ok {
			value := errorDetail.ErrorDetail()
			if value.Location != "" {
				return value.Location + ": " + value.Message
			}
			if value.Message != "" {
				return value.Message
			}
		}
	}
	return message
}
