package health_check

import (
	"context"
	"fmt"

	"github.com/jasonlabz/generate-example-project/server/manager/health_check"
)

type readinessCheckServiceImpl struct {
	manager health_check.ReadinessCheckManager
}

var _ ReadinessCheckService = (*readinessCheckServiceImpl)(nil)

// NewReadinessCheckService creates a ReadinessCheckService backed by manager.
func NewReadinessCheckService(manager health_check.ReadinessCheckManager) ReadinessCheckService {
	return &readinessCheckServiceImpl{manager: manager}
}

// Check reports a readiness dependency failure with service-level context.
func (s *readinessCheckServiceImpl) Check(ctx context.Context) (ReadinessCheckResult, error) {
	if err := s.manager.Check(ctx); err != nil {
		return ReadinessCheckResult{}, fmt.Errorf("check readiness: %w", err)
	}

	return ReadinessCheckResult{Status: "ready"}, nil
}
