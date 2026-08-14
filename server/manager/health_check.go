// Package manager contains application managers that coordinate infrastructure work.
package manager

import "context"

// HealthCheckManager coordinates the health-check dependencies.
type HealthCheckManager interface {
	Check(ctx context.Context) error
}

// HealthProbe checks the availability of one health-check dependency.
type HealthProbe interface {
	Probe(ctx context.Context) error
}
