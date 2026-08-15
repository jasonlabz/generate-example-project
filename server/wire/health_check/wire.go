// Package health_check assembles the health-check production dependency graph.
package health_check

import (
	health_check_controller "github.com/jasonlabz/generate-example-project/server/controller/health_check"
	health_check_manager "github.com/jasonlabz/generate-example-project/server/manager/health_check"
	health_check_service "github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// NewHealthCheckController assembles the production health-check controller.
func NewHealthCheckController() health_check_controller.HealthCheckController {
	probe := health_check_manager.NewLocalProbe()
	manager := health_check_manager.NewManager(probe)
	service := health_check_service.NewService(manager)

	return health_check_controller.NewHealthCheckController(service)
}
