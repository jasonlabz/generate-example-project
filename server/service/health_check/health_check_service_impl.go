package health_check

import (
	"context"
	"fmt"

	"github.com/jasonlabz/generate-example-project/server/manager/health_check"
)

type healthCheckServiceImpl struct {
	manager health_check.HealthCheckManager
}

var _ HealthCheckService = (*healthCheckServiceImpl)(nil)

// NewHealthCheckService creates a HealthCheckService backed by manager.
func NewHealthCheckService(manager health_check.HealthCheckManager) HealthCheckService {
	return &healthCheckServiceImpl{manager: manager}
}

// Check reports a manager failure with service-level context.
func (s *healthCheckServiceImpl) Check(ctx context.Context) (HealthCheckResult, error) {
	if err := s.manager.Check(ctx); err != nil {
		return HealthCheckResult{}, fmt.Errorf("check health: %w", err)
	}

	return HealthCheckResult{Status: "success"}, nil
}
