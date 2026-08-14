package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

type healthCheckOutput struct {
	Body struct {
		Code        int      `json:"code"`
		Version     string   `json:"version"`
		CurrentTime string   `json:"current_time"`
		Data        []string `json:"data"`
	}
}

// RegisterHealthCheck registers the typed health-check operation with Huma.
func RegisterHealthCheck(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health-check",
		Summary:     "Health check",
		Tags:        []string{"Health check"},
	}, func(ctx context.Context, _ *struct{}) (*healthCheckOutput, error) {
		output := &healthCheckOutput{}
		output.Body.Code = 0
		output.Body.Version = "v1"
		output.Body.CurrentTime = time.Now().Format(time.DateTime)
		output.Body.Data = []string{health_check.GetService().DoCheck(ctx)}
		return output, nil
	})
}
