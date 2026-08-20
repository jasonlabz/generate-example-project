// Package health_check coordinates dependencies used by the health-check use case.
package health_check

import "context"

// Manager coordinates the dependencies used by health-check use cases.
type Manager interface {
	Check(ctx context.Context) error
	CheckReadiness(ctx context.Context) error
}

// HealthProbe checks the availability of one health-check dependency.
type HealthProbe interface {
	Probe(ctx context.Context) error
}
