package health_check

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jasonlabz/generate-example-project/common/consts"
	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
)

type readinessCheckControllerImpl struct {
	service health_check.ReadinessCheckService
}

var _ ReadinessCheckController = (*readinessCheckControllerImpl)(nil)

// NewReadinessCheckController constructs the readiness-check controller.
func NewReadinessCheckController(readinessCheckService health_check.ReadinessCheckService) ReadinessCheckController {
	return &readinessCheckControllerImpl{service: readinessCheckService}
}

// Register registers GET /readiness-check on api.
func (c *readinessCheckControllerImpl) Register(api huma.API) {
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
	}, c.handle)
}

// handle converts readiness results to the existing HTTP response contract.
func (c *readinessCheckControllerImpl) handle(ctx context.Context, _ *struct{}) (*healthCheckOutput, error) {
	result, err := c.service.Check(ctx)
	if err != nil {
		return nil, humax.InternalServerError(consts.APIVersionV1, err)
	}

	return toReadinessCheckOutput(result), nil
}
