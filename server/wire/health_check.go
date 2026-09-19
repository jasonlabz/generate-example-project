package wire

import (
	healthcheckcontroller "github.com/jasonlabz/generate-example-project/server/controller/health_check"
	healthcheckmanager "github.com/jasonlabz/generate-example-project/server/manager/health_check"
	healthcheckservice "github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// NewHealthCheckController 组装 health-check 模块的生产依赖图。
//
// 依赖自下而上连接：Probe -> Manager -> Service -> Controller。
func NewHealthCheckController() *healthcheckcontroller.Controller {
	probe := healthcheckmanager.NewLocalProbe()
	checkManager := healthcheckmanager.NewManager(probe)
	checkService := healthcheckservice.NewService(checkManager)

	return healthcheckcontroller.NewController(checkService)
}
