package health_check

import "context"

type localProbe struct{}

var _ HealthProbe = (*localProbe)(nil)

// NewLocalProbe creates the local health probe.
func NewLocalProbe() HealthProbe {
	return &localProbe{}
}

// Probe confirms the example application is available without external I/O.
func (p *localProbe) Probe(_ context.Context) error {
	return nil
}
