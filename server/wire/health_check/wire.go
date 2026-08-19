// Package health_check assembles the health-check production dependency graph.
package health_check

import (
	controller "github.com/jasonlabz/generate-example-project/server/controller/health_check"
	manager "github.com/jasonlabz/generate-example-project/server/manager/health_check"
	service "github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// NewHealthCheckController assembles the production liveness-check controller.
func NewHealthCheckController() controller.HealthCheckController {
	probe := manager.NewLocalProbe()
	checkManager := manager.NewHealthCheckManager(probe)
	checkService := service.NewHealthCheckService(checkManager)

	return controller.NewHealthCheckController(checkService)
}

// NewReadinessCheckController assembles the production readiness-check controller.
func NewReadinessCheckController() controller.ReadinessCheckController {
	probe := manager.NewLocalProbe()
	checkManager := manager.NewReadinessCheckManager(probe)
	checkService := service.NewReadinessCheckService(checkManager)

	return controller.NewReadinessCheckController(checkService)
}
