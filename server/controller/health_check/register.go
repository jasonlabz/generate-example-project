// Package health_check is the HTTP controller for the health-check domain.
//
// Controllers own Huma operation registration, HTTP DTOs, and conversions between
// HTTP and service models. They do not contain business logic.
package health_check

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Register registers all HTTP operations maintained by the health-check Controller.
func (c *Controller) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "health-check",
		Method:        http.MethodGet,
		Path:          "/health-check",
		Summary:       "健康检查",
		Description:   "用于负载均衡器、容器探针和运维监控检查服务是否存活。成功时返回 code=0、data=[\"success\"]；依赖检查失败时返回 500 及统一错误信封。",
		Tags:          []string{"系统", "健康检查"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusInternalServerError},
		Responses: map[string]*huma.Response{
			"200": {Description: "服务正常，data 包含当前健康状态。"},
		},
	}, c.handleHealthCheck)

	huma.Register(api, huma.Operation{
		OperationID:   "readiness-check",
		Method:        http.MethodGet,
		Path:          "/readiness-check",
		Summary:       "就绪检查",
		Description:   "用于确认服务是否已准备好接收流量。",
		Tags:          []string{"系统", "健康检查"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusInternalServerError},
		Responses: map[string]*huma.Response{
			"200": {Description: "服务已就绪，data 包含当前就绪状态。"},
		},
	}, c.handleReadinessCheck)
}
