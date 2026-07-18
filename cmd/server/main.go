package main

import (
	"context"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
	"github.com/jasonlabz/generate-example-project/internal/server"
)

// @title			generate-example-project
// @version		1.0
// @description	基于 Gin 的标准项目模板
// @contact.name	your name
// @contact.email	mail_name@qq.com
// @BasePath		/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bootstrap.MustInit(ctx)
	server.Run(ctx)
}
