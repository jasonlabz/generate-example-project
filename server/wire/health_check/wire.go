// Package health_check assembles the health-check production dependency graph.
package health_check

import (
	controller "github.com/jasonlabz/generate-example-project/server/controller/health_check"
	manager "github.com/jasonlabz/generate-example-project/server/manager/health_check"
	service "github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// NewController assembles the production health-check controller.
func NewController() controller.HealthCheckController {
	_probe := manager.NewLocalProbe()
	_manager := manager.NewManager(_probe)
	_service := service.NewService(_manager)

	return controller.NewHealthCheckController(_service)
}
