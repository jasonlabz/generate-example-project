package controller

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/service"
)

// HealthCheckController registers the health-check HTTP operation.
type HealthCheckController interface {
	Register(api huma.API)
}

// Controller adapts the health-check service to Huma.
type Controller struct {
	service service.HealthCheckService
}

var _ HealthCheckController = (*Controller)(nil)

// NewHealthCheckController creates a health-check controller backed by service.
func NewHealthCheckController(service service.HealthCheckService) HealthCheckController {
	return &Controller{service: service}
}

// Register adds the typed health-check operation to api.
func (c *Controller) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health-check",
		Summary:     "Health check",
		Tags:        []string{"Health check"},
	}, func(ctx context.Context, _ *struct{}) (*humax.Output[[]string], error) {
		if err := c.service.Check(ctx); err != nil {
			return nil, humax.InternalServerError("v1", err)
		}

		return humax.Success("v1", []string{"success"}), nil
	})
}
