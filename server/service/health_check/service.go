package health_check

import (
	"context"
	"fmt"

	health_check_manager "github.com/jasonlabz/generate-example-project/server/manager/health_check"
)

type service struct {
	manager health_check_manager.HealthCheckManager
}

var _ HealthCheckService = (*service)(nil)

// NewService creates a HealthCheckService backed by manager.
func NewService(manager health_check_manager.HealthCheckManager) HealthCheckService {
	return &service{manager: manager}
}

// Check reports a manager failure with service-level context.
func (s *service) Check(ctx context.Context) (HealthCheckResult, error) {
	if err := s.manager.Check(ctx); err != nil {
		return HealthCheckResult{}, fmt.Errorf("check health: %w", err)
	}

	return HealthCheckResult{Status: "success"}, nil
}
