// Package user is the HTTP controller for the user domain.
//
// Controllers own Huma operation registration, HTTP DTOs, and conversions between
// HTTP and service models. They do not contain business logic.
package user

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jasonlabz/generate-example-project/common/consts"
	"github.com/jasonlabz/generate-example-project/common/humax"
)

// Register registers all HTTP operations maintained by the user Controller.
//
// Errors 声明的是该接口实际可能返回的状态码：业务失败会带上 apperr 目录中登记的
// HTTP 语义（如 100004001 → 404），只有未预期的故障才是 500。声明齐全才能让
// OpenAPI 文档与真实行为一致。
func (c *Controller) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "user-get",
		Method:        http.MethodGet,
		Path:          "/users/{id}",
		Summary:       "查询用户",
		Description:   "按主键查询用户；用户不存在时返回 404 与业务 code 100004001。",
		Tags:          []string{"用户"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusNotFound, http.StatusInternalServerError},
	}, humax.Wrap(consts.APIVersionV1, c.handleGet))

	huma.Register(api, huma.Operation{
		OperationID:   "user-list",
		Method:        http.MethodGet,
		Path:          "/users",
		Summary:       "分页查询用户",
		Description:   "按关键字筛选并分页返回；keyword 不传表示不筛选。分页信息随 pagination 字段返回。",
		Tags:          []string{"用户"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusInternalServerError},
	}, humax.WrapPage(consts.APIVersionV1, c.handleList))

	huma.Register(api, huma.Operation{
		OperationID:   "user-create",
		Method:        http.MethodPost,
		Path:          "/users",
		Summary:       "创建用户",
		Description:   "用户名全局唯一；重名时返回 409 与业务 code 100005001。",
		Tags:          []string{"用户"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusConflict, http.StatusInternalServerError},
	}, humax.Wrap(consts.APIVersionV1, c.handleCreate))

	huma.Register(api, huma.Operation{
		OperationID:   "user-delete",
		Method:        http.MethodDelete,
		Path:          "/users/{id}",
		Summary:       "删除用户",
		Description:   "用户不存在时返回 404 与业务 code 100004001；重复删除同一 ID 结果一致。",
		Tags:          []string{"用户"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusNotFound, http.StatusInternalServerError},
	}, humax.Wrap(consts.APIVersionV1, c.handleDelete))

	huma.Register(api, huma.Operation{
		OperationID:   "user-export",
		Method:        http.MethodGet,
		Path:          "/users/export",
		Summary:       "导出用户",
		Description:   "以文件流返回用户列表（CSV）。格式暂不支持 xlsx，传入时返回 422 与业务 code 100006001。",
		Tags:          []string{"用户"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnprocessableEntity, http.StatusInternalServerError},
		Responses: map[string]*huma.Response{
			"200": humax.BinaryResponse("用户列表 CSV 文件", "text/csv"),
		},
	}, c.handleExport)
}
