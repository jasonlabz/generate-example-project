package health_check

import "github.com/jasonlabz/generate-example-project/common/humax"

// healthCheckOutput 是健康检查接口的 HTTP 响应模型。
//
// huma 响应模型约定：结构体嵌入 Body 字段作为响应体；有参数时入参结构体
// 用 huma tag 声明位置与校验（如 `path:"id" example:"1"`），huma 据此生成
// OpenAPI 参数与响应 schema。
type healthCheckOutput struct {
	Body *humax.Envelope[[]string]
}
