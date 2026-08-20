package health_check

import (
	"github.com/jasonlabz/generate-example-project/common/consts"
	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// toHealthCheckOutput converts a service Result into the shared HTTP response model.
func toHealthCheckOutput(result health_check.Result) *healthCheckOutput {
	return &healthCheckOutput{Body: humax.New(consts.APIVersionV1, []string{result.Status})}
}
