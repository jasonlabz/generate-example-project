// Package humax adapts common API response envelopes for Huma handlers.
package humax

import (
	"errors"
	"net/http"

	"github.com/jasonlabz/generate-example-project/common/response"
)

// Output is a typed Huma response body.
type Output[T any] struct {
	Body *response.Envelope[T]
}

// Success creates a typed successful response for a Huma handler.
func Success[T any](version string, data T) *Output[T] {
	return &Output[T]{
		Body: response.New(version, data),
	}
}

// Error is a status-aware common response envelope for a Huma handler.
type Error struct {
	*response.Envelope[[]any]
	status int
	cause  error
}

// InternalServerError adapts an unexpected error to the existing 500 response contract.
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

// Error returns the cause text for the Go error interface.
func (e *Error) Error() string {
	return e.cause.Error()
}

// GetStatus returns the HTTP status for Huma's StatusError interface.
func (e *Error) GetStatus() int {
	return e.status
}
