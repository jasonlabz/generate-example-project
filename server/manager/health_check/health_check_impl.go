// Package health_check implements the health-check manager and its local probe.
package health_check

import (
	"context"
	"fmt"

	"github.com/jasonlabz/generate-example-project/server/manager"
)

// Manager delegates health checks to one infrastructure probe.
type Manager struct {
	probe manager.HealthProbe
}

var _ manager.HealthCheckManager = (*Manager)(nil)

// NewManager creates a HealthCheckManager backed by probe.
func NewManager(probe manager.HealthProbe) manager.HealthCheckManager {
	return &Manager{probe: probe}
}

// Check reports a probe failure with manager-level context.
func (m *Manager) Check(ctx context.Context) error {
	if err := m.probe.Probe(ctx); err != nil {
		return fmt.Errorf("probe health: %w", err)
	}

	return nil
}

// LocalProbe is the deterministic health probe used by the example application.
type LocalProbe struct{}

var _ manager.HealthProbe = (*LocalProbe)(nil)

// NewLocalProbe creates the local health probe.
func NewLocalProbe() manager.HealthProbe {
	return &LocalProbe{}
}

// Probe confirms the example application is available without external I/O.
func (p *LocalProbe) Probe(_ context.Context) error {
	return nil
}
