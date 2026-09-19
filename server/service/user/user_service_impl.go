package user

import (
	"context"
	"fmt"

	"github.com/jasonlabz/generate-example-project/common/apperr"
	"github.com/jasonlabz/generate-example-project/server/manager/user"
)

type serviceImpl struct {
	manager user.Manager
}

var _ Service = (*serviceImpl)(nil)

// NewService creates a Service backed by manager.
func NewService(manager user.Manager) Service {
	return &serviceImpl{manager: manager}
}

// Get 按主键查询用户。
// 记录不存在不是技术故障，而是可预期的业务结果，因此转换为 apperr.NotFound，
// 由 humax 统一映射为 HTTP 404 + 业务 code，而不是降级成 500。
func (s *serviceImpl) Get(ctx context.Context, id int64) (User, error) {
	record, exists, err := s.manager.FindByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("find user %d: %w", id, err)
	}
	if !exists {
		return User{}, apperr.NotFound.WithMessage(fmt.Sprintf("用户 %d 不存在", id))
	}
	return toUser(record), nil
}

// List 按关键字分页查询用户。
func (s *serviceImpl) List(ctx context.Context, keyword string, offset, limit int64) ([]User, int64, error) {
	records, total, err := s.manager.Search(ctx, keyword, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("search users: %w", err)
	}
	return toUsers(records), total, nil
}

// Create 创建用户，用户名必须唯一。
func (s *serviceImpl) Create(ctx context.Context, name, email string) (User, error) {
	_, exists, err := s.manager.FindByName(ctx, name)
	if err != nil {
		return User{}, fmt.Errorf("check user name %q: %w", name, err)
	}
	if exists {
		return User{}, apperr.Conflict.WithMessage(fmt.Sprintf("用户名 %s 已存在", name))
	}

	record, err := s.manager.Insert(ctx, user.Record{Name: name, Email: email})
	if err != nil {
		return User{}, fmt.Errorf("insert user %q: %w", name, err)
	}
	return toUser(record), nil
}

// Delete 删除用户，用户不存在时返回 NotFound。
func (s *serviceImpl) Delete(ctx context.Context, id int64) error {
	deleted, err := s.manager.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	if !deleted {
		return apperr.NotFound.WithMessage(fmt.Sprintf("用户 %d 不存在", id))
	}
	return nil
}

// Export 返回待导出的用户列表；列表为空时视为不符合导出业务规则。
func (s *serviceImpl) Export(ctx context.Context) ([]User, error) {
	// limit 传 0 表示不限制条数，导出不分页。
	records, _, err := s.manager.Search(ctx, "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("search users for export: %w", err)
	}
	if len(records) == 0 {
		return nil, apperr.BusinessRule.WithMessage("暂无用户可导出")
	}
	return toUsers(records), nil
}

// toUser 把存储模型转换为业务模型。
func toUser(record user.Record) User {
	return User{ID: record.ID, Name: record.Name, Email: record.Email}
}

// toUsers 批量转换存储模型。
func toUsers(records []user.Record) []User {
	users := make([]User, 0, len(records))
	for _, record := range records {
		users = append(users, toUser(record))
	}
	return users
}
