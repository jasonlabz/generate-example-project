package health_check

import (
	"context"
	"fmt"
)

type healthCheckManagerImpl struct {
	probe HealthProbe
}

var _ HealthCheckManager = (*healthCheckManagerImpl)(nil)

// NewHealthCheckManager creates a HealthCheckManager backed by probe.
func NewHealthCheckManager(probe HealthProbe) HealthCheckManager {
	return &healthCheckManagerImpl{probe: probe}
}

// Check reports a probe failure with manager-level context.
func (m *healthCheckManagerImpl) Check(ctx context.Context) error {
	if err := m.probe.Probe(ctx); err != nil {
		return fmt.Errorf("probe health: %w", err)
	}

	return nil
}
