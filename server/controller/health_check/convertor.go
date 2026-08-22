package health_check

import "github.com/jasonlabz/generate-example-project/server/service/health_check"

// toHealthCheckData converts a service Result into controller response data.
func toHealthCheckData(result health_check.Result) *[]string {
	data := []string{result.Status}
	return &data
}
