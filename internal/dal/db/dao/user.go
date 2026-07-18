package dao

import (
	"context"

	"gorm.io/gorm"

	"github.com/jasonlabz/generate-example-project/internal/dal/db/model"
)

// UserDao 示例 DAO：演示构造装载模式，业务项目可删除。
type UserDao struct {
	db *gorm.DB
}

// NewUserDao 构造指定连接的 UserDao，单测中传入内存库。
func NewUserDao(db *gorm.DB) *UserDao { return &UserDao{db: db} }

// GetUserDao 返回使用默认装载连接的 UserDao，业务代码使用。
func GetUserDao() *UserDao { return NewUserDao(defaultDB) }

func (d *UserDao) Insert(ctx context.Context, user *model.User) error {
	return d.db.WithContext(ctx).Create(user).Error
}

func (d *UserDao) SelectByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	if err := d.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *UserDao) UpdateName(ctx context.Context, id int64, name string) error {
	return d.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).Update("name", name).Error
}

func (d *UserDao) DeleteByID(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.User{}, id).Error
}
