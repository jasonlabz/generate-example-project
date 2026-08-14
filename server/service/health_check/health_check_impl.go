// Package health_check implements the health-check service.
package health_check

import (
	"context"
	"fmt"

	"github.com/jasonlabz/generate-example-project/server/manager"
	"github.com/jasonlabz/generate-example-project/server/service"
)

// Service applies the application policy for the health check.
type Service struct {
	manager manager.HealthCheckManager
}

var _ service.HealthCheckService = (*Service)(nil)

// NewService creates a HealthCheckService backed by manager.
func NewService(manager manager.HealthCheckManager) service.HealthCheckService {
	return &Service{manager: manager}
}

// Check reports a manager failure with service-level context.
func (s *Service) Check(ctx context.Context) error {
	if err := s.manager.Check(ctx); err != nil {
		return fmt.Errorf("check health: %w", err)
	}

	return nil
}
