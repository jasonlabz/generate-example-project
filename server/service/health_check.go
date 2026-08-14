package service

import "context"

// HealthCheckService performs the application health check.
type HealthCheckService interface {
	Check(ctx context.Context) error
}
