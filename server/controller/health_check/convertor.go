package health_check

import (
	"github.com/jasonlabz/generate-example-project/common/response"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// toHealthCheckOutput 把业务层结果（service.HealthCheckResult）转换为
// HTTP 响应模型（healthCheckOutput）。DTO 与业务模型分离，转换逻辑
// 集中在 convertor 文件，控制器 handle 保持简洁。
func toHealthCheckOutput(result health_check.HealthCheckResult) *healthCheckOutput {
	return &healthCheckOutput{Body: response.New(apiVersion, []string{result.Status})}
}
