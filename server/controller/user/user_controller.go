package user

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jasonlabz/generate-example-project/common/apperr"
	"github.com/jasonlabz/generate-example-project/common/consts"
	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/service/user"
)

// Controller exposes the user domain HTTP operations.
type Controller struct {
	service user.Service
}

// NewController constructs a Controller with its service dependency.
func NewController(service user.Service) *Controller {
	return &Controller{service: service}
}

// handleGet 查询单个用户。service 已把"不存在"转换为 apperr.NotFound，
// 这里只需原样上抛，由 humax.Wrap 统一映射为 HTTP 404 + 业务 code。
func (c *Controller) handleGet(ctx context.Context, in *getUserInput) (*userVO, error) {
	item, err := c.service.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}

	view := toUserVO(item)
	return &view, nil
}

// handleList 分页查询用户，分页信息由 humax.WrapPage 统一封装进响应信封。
func (c *Controller) handleList(ctx context.Context, in *listUsersInput) ([]userVO, *humax.Pagination, error) {
	pagination := &humax.Pagination{Page: in.Page, PageSize: in.PageSize}

	items, total, err := c.service.List(ctx, in.Keyword, pagination.GetOffset(), in.PageSize)
	if err != nil {
		return nil, nil, err
	}
	pagination.Total = total
	pagination.GetPageCount()

	return toUserVOs(items), pagination, nil
}

// handleCreate 创建用户。参数必填与格式由 huma 校验，重名由 service 返回 apperr.Conflict。
func (c *Controller) handleCreate(ctx context.Context, in *createUserInput) (*userVO, error) {
	item, err := c.service.Create(ctx, in.Body.Name, in.Body.Email)
	if err != nil {
		return nil, err
	}

	view := toUserVO(item)
	return &view, nil
}

// handleDelete 删除用户。
func (c *Controller) handleDelete(ctx context.Context, in *deleteUserInput) (*userVO, error) {
	if err := c.service.Delete(ctx, in.ID); err != nil {
		return nil, err
	}
	return &userVO{ID: in.ID}, nil
}

// handleExport 导出用户为文件流。
//
// xlsx 虽然通过了 huma 的 enum 校验，但当前并未实现，属于"参数合法但业务不支持"的
// 场景——这类失败用 humax.BusinessError 表达最合适：它不会退回 RFC7807，
// 也无需为它单独登记一个 apperr 错误码。
func (c *Controller) handleExport(ctx context.Context, in *exportUsersInput) (*huma.StreamResponse, error) {
	if in.Format == "xlsx" {
		return nil, humax.BusinessError(consts.APIVersionV1, apperr.BusinessRule.Code(), "暂不支持 xlsx 导出，请使用 csv")
	}

	items, err := c.service.Export(ctx)
	if err != nil {
		// 文件流接口不走 humax.Wrap，错误不会经过 mapError，需显式上报给 ErrorReporter。
		return nil, humax.ReportError(ctx, err)
	}

	return humax.File(consts.APIVersionV1, &humax.FileDownloadConfig{
		Filename:    "users.csv",
		ContentType: "text/csv",
		Content:     toCSV(toUserVOs(items)),
	})
}
