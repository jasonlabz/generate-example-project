package health_check

import (
	"context"

	"github.com/jasonlabz/generate-example-project/common/consts"
	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// Controller exposes the health-check domain HTTP operations.
type Controller struct {
	service health_check.Service
}

// NewController constructs a Controller with its service dependency.
func NewController(service health_check.Service) *Controller {
	return &Controller{service: service}
}

// handleHealthCheck converts the liveness use case to the HTTP response contract.
func (c *Controller) handleHealthCheck(ctx context.Context, _ *struct{}) (*healthCheckOutput, error) {
	result, err := c.service.Check(ctx)
	if err != nil {
		return nil, humax.InternalServerError(consts.APIVersionV1, err)
	}

	return toHealthCheckOutput(result), nil
}

// handleReadinessCheck converts the readiness use case to the HTTP response contract.
func (c *Controller) handleReadinessCheck(ctx context.Context, _ *struct{}) (*healthCheckOutput, error) {
	result, err := c.service.CheckReadiness(ctx)
	if err != nil {
		return nil, humax.InternalServerError(consts.APIVersionV1, err)
	}

	return toHealthCheckOutput(result), nil
}
