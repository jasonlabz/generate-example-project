// Package health_check exposes the health-check HTTP operation.
package health_check

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// HealthCheckController registers the health-check HTTP operation.
type HealthCheckController interface {
	Register(api huma.API)
}

// Register adds the typed health-check operation to api.
func (c *controller) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health-check",
		Summary:     "Health check",
		Tags:        []string{"Health check"},
	}, c.handle)
}
