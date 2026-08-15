package health_check

import "github.com/jasonlabz/generate-example-project/common/response"

const apiVersion = "v1"

type healthCheckOutput struct {
	Body *response.Envelope[[]string]
}
