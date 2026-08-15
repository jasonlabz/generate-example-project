package health_check

import (
	"context"
	"fmt"
)

type manager struct {
	probe HealthProbe
}

var _ HealthCheckManager = (*manager)(nil)

// NewManager creates a HealthCheckManager backed by probe.
func NewManager(probe HealthProbe) HealthCheckManager {
	return &manager{probe: probe}
}

// Check reports a probe failure with manager-level context.
func (m *manager) Check(ctx context.Context) error {
	if err := m.probe.Probe(ctx); err != nil {
		return fmt.Errorf("probe health: %w", err)
	}

	return nil
}
