package dao

import "gorm.io/gorm"

var defaultDB *gorm.DB

// SetGormDB 装载默认数据库连接，bootstrap 初始化 DB 后调用一次。
func SetGormDB(db *gorm.DB) { defaultDB = db }

// DefaultDB 返回已装载的默认连接。
func DefaultDB() *gorm.DB { return defaultDB }
