// Package health_check implements the health-check application use case.
package health_check

import "context"

// HealthCheckService performs the application health check.
type HealthCheckService interface {
	Check(ctx context.Context) (HealthCheckResult, error)
}

// ReadinessCheckService verifies whether the application can accept traffic.
type ReadinessCheckService interface {
	Check(ctx context.Context) (ReadinessCheckResult, error)
}
