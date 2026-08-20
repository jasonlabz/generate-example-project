package health_check

import (
	"context"
	"fmt"

	"github.com/jasonlabz/generate-example-project/server/manager/health_check"
)

type serviceImpl struct {
	manager health_check.Manager
}

var _ Service = (*serviceImpl)(nil)

// NewService creates a Service backed by manager.
func NewService(manager health_check.Manager) Service {
	return &serviceImpl{manager: manager}
}

// Check reports a liveness dependency failure with service-level context.
func (s *serviceImpl) Check(ctx context.Context) (Result, error) {
	if err := s.manager.Check(ctx); err != nil {
		return Result{}, fmt.Errorf("check health: %w", err)
	}

	return Result{Status: "success"}, nil
}

// CheckReadiness reports a readiness dependency failure with service-level context.
func (s *serviceImpl) CheckReadiness(ctx context.Context) (Result, error) {
	if err := s.manager.CheckReadiness(ctx); err != nil {
		return Result{}, fmt.Errorf("check readiness: %w", err)
	}

	return Result{Status: "ready"}, nil
}
