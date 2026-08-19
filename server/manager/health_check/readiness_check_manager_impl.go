package health_check

import (
	"context"
	"fmt"
)

type readinessCheckManagerImpl struct {
	probe HealthProbe
}

var _ ReadinessCheckManager = (*readinessCheckManagerImpl)(nil)

// NewReadinessCheckManager creates a ReadinessCheckManager backed by probe.
func NewReadinessCheckManager(probe HealthProbe) ReadinessCheckManager {
	return &readinessCheckManagerImpl{probe: probe}
}

// Check reports a probe failure with readiness-manager context.
func (m *readinessCheckManagerImpl) Check(ctx context.Context) error {
	if err := m.probe.Probe(ctx); err != nil {
		return fmt.Errorf("probe readiness: %w", err)
	}

	return nil
}
