package health_check

import (
	"context"

	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

type controller struct {
	service health_check.HealthCheckService
}

var _ HealthCheckController = (*controller)(nil)

// NewHealthCheckController creates a health-check controller backed by healthCheckService.
func NewHealthCheckController(healthCheckService health_check.HealthCheckService) HealthCheckController {
	return &controller{service: healthCheckService}
}

// handle adapts the service result and failure to the existing HTTP contract.
func (c *controller) handle(ctx context.Context, _ *struct{}) (*healthCheckOutput, error) {
	result, err := c.service.Check(ctx)
	if err != nil {
		return nil, humax.InternalServerError(apiVersion, err)
	}

	return toHealthCheckOutput(result), nil
}
