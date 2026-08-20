// Package health_check assembles the health-check production dependency graph.
package health_check

import (
	controller "github.com/jasonlabz/generate-example-project/server/controller/health_check"
	manager "github.com/jasonlabz/generate-example-project/server/manager/health_check"
	service "github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// NewController assembles the production health-check Controller.
func NewController() *controller.Controller {
	probe := manager.NewLocalProbe()
	checkManager := manager.NewManager(probe)
	checkService := service.NewService(checkManager)

	return controller.NewController(checkService)
}
