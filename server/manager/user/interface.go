// Package user 提供用户数据的访问能力（技术能力层）。
//
// manager 只暴露"取数/写数"这一类技术操作，不做任何业务判断：
// "用户名重复算不算错""不存在要不要报错"都属于 service 的职责。
package user

import "context"

// Record 是用户数据的存储模型，字段与存储结构一一对应。
type Record struct {
	ID    int64
	Name  string
	Email string
}

// Manager 提供用户数据的访问能力。
// 各方法的 bool 返回值表示"记录是否存在"，与 error（技术故障）严格区分：
// 查不到是正常业务结果，连接失败才是错误。
type Manager interface {
	// FindByID 按主键查询用户。
	FindByID(ctx context.Context, id int64) (Record, bool, error)
	// FindByName 按用户名查询用户，用于唯一性校验。
	FindByName(ctx context.Context, name string) (Record, bool, error)
	// Search 按关键字分页查询；keyword 为空表示不筛选，limit<=0 表示不限制条数。
	// 第二个返回值是与筛选条件匹配的总条数（不受分页影响）。
	Search(ctx context.Context, keyword string, offset, limit int64) ([]Record, int64, error)
	// Insert 写入用户并返回分配主键后的记录。
	Insert(ctx context.Context, record Record) (Record, error)
	// Delete 删除用户，bool 表示是否真的删到了记录。
	Delete(ctx context.Context, id int64) (bool, error)
}
