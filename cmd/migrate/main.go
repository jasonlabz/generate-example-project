package main

import (
	"context"
	"log"

	"github.com/jasonlabz/generate-example-project/bootstrap"
)

// migrate 命令独立执行数据库迁移（建库 → DDL → seed）后退出，
// 与服务启动时 MustInit 内嵌的迁移步骤使用同一套实现，结果等价。
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("migrate command started")
	bootstrap.MustMigrate(ctx)
	log.Println("migrate command finished")
}
