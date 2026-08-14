// Package response provides framework-neutral API response envelopes.
package response

import "time"

// Envelope is the common JSON response body shared by HTTP adapters.
type Envelope[T any] struct {
	Code        int    `json:"code"`
	Message     string `json:"message,omitempty"`
	ErrTrace    string `json:"err_trace,omitempty"`
	Version     string `json:"version"`
	CurrentTime string `json:"current_time"`
	Data        T      `json:"data"`
}

// New creates a successful response envelope with the current local time.
func New[T any](version string, data T) *Envelope[T] {
	return &Envelope[T]{
		Version:     version,
		CurrentTime: time.Now().Format(time.DateTime),
		Data:        data,
	}
}

// NewError creates an error response envelope with the current local time.
func NewError[T any](version string, data T, code int, message, trace string) *Envelope[T] {
	envelope := New(version, data)
	envelope.Code = code
	envelope.Message = message
	envelope.ErrTrace = trace

	return envelope
}
