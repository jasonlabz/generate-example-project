package health_check

import (
	"github.com/jasonlabz/generate-example-project/common/response"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// toHealthCheckOutput converts the service result into the HTTP response model.
func toHealthCheckOutput(result health_check.HealthCheckResult) *healthCheckOutput {
	return &healthCheckOutput{Body: response.New(apiVersion, []string{result.Status})}
}
