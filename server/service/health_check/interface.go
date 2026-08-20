// Package health_check implements the health-check application use case.
package health_check

import "context"

// Service provides the health-check use cases for the HTTP layer.
type Service interface {
	Check(ctx context.Context) (Result, error)
	CheckReadiness(ctx context.Context) (Result, error)
}
