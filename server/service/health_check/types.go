package health_check

// HealthCheckResult describes the health-check state for service consumers.
type HealthCheckResult struct {
	Status string
}

// ReadinessCheckResult describes the readiness state for service consumers.
type ReadinessCheckResult struct {
	Status string
}
