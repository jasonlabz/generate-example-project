package dao

import "gorm.io/gorm"

var defaultDB *gorm.DB

// SetGormDB 装载默认数据库连接，bootstrap 初始化 DB 后调用一次。
func SetGormDB(db *gorm.DB) { defaultDB = db }

// DefaultDB 返回已装载的默认连接，未装载时直接 panic 便于尽早暴露配置问题。
func DefaultDB() *gorm.DB {
	if defaultDB == nil {
		panic("dao: defaultDB 未装载，请确认 datasource.enable=true 且 bootstrap.MustInit 已执行")
	}
	return defaultDB
}
