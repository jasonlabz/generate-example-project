package health_check

import (
	"context"

	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

// controller 是健康检查模块的控制器实现，持有业务层（service）依赖。
type controller struct {
	service health_check.HealthCheckService
}

// 编译期断言：controller 必须实现 HealthCheckController 接口。
var _ HealthCheckController = (*controller)(nil)

// NewHealthCheckController 构造健康检查控制器（依赖注入业务层服务）。
// 业务模块控制器统一采用「接口 + 私有实现 + 构造函数」形态，便于 mock 与测试。
func NewHealthCheckController(healthCheckService health_check.HealthCheckService) HealthCheckController {
	return &controller{service: healthCheckService}
}

// handle 是 huma 操作处理器：把业务层结果/失败转换为既有 HTTP 响应契约。
//
// huma handler 签名约定：func(ctx context.Context, input *In) (*Out, error)
//   - 入参 *struct{}：无请求参数（有参数时定义结构体并用 huma tag 声明位置）；
//   - 出参 *healthCheckOutput：响应体（Body 字段包装统一信封 response.Envelope）；
//   - error：nil 表示成功；返回实现 huma.StatusError 接口（GetStatus() int）
//     的错误可携带状态码（见 common/humax.Error）。
func (c *controller) handle(ctx context.Context, _ *struct{}) (*healthCheckOutput, error) {
	result, err := c.service.Check(ctx)
	if err != nil {
		// 业务错误统一转换为 500 信封响应，保持与旧接口一致的 HTTP 契约。
		return nil, humax.InternalServerError(apiVersion, err)
	}

	return toHealthCheckOutput(result), nil
}
