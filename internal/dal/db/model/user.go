package model

import "time"

// User 示例模型：演示 gentol 生成物的落点与风格，业务项目可删除。
type User struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;size:64;not null" json:"name"`
	Email     string    `gorm:"column:email;size:128" json:"email"`
	Status    int       `gorm:"column:status;default:0" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "example_user" }
