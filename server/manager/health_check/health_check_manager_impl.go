package health_check

import (
	"context"
	"fmt"
)

type managerImpl struct {
	probe HealthProbe
}

var _ Manager = (*managerImpl)(nil)

// NewManager creates a Manager backed by probe.
func NewManager(probe HealthProbe) Manager {
	return &managerImpl{probe: probe}
}

// Check reports a liveness probe failure with manager-level context.
func (m *managerImpl) Check(ctx context.Context) error {
	if err := m.probe.Probe(ctx); err != nil {
		return fmt.Errorf("probe health: %w", err)
	}

	return nil
}

// CheckReadiness reports a readiness probe failure with manager-level context.
func (m *managerImpl) CheckReadiness(ctx context.Context) error {
	if err := m.probe.Probe(ctx); err != nil {
		return fmt.Errorf("probe readiness: %w", err)
	}

	return nil
}
