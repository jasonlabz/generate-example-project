// Package user 编排用户领域的业务用例。
package user

import "context"

// Service 暴露用户领域的用例，供 controller 调用。
type Service interface {
	// Get 按主键查询用户；不存在时返回 apperr.NotFound 语义的错误。
	Get(ctx context.Context, id int64) (User, error)
	// List 按关键字分页查询，返回当页数据与匹配总数。
	List(ctx context.Context, keyword string, offset, limit int64) ([]User, int64, error)
	// Create 创建用户；用户名重复时返回 apperr.Conflict 语义的错误。
	Create(ctx context.Context, name, email string) (User, error)
	// Delete 删除用户；不存在时返回 apperr.NotFound 语义的错误。
	Delete(ctx context.Context, id int64) error
	// Export 返回待导出的用户列表；无数据时返回 apperr.BusinessRule 语义的错误。
	Export(ctx context.Context) ([]User, error)
}
