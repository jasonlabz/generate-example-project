package router

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/jasonlabz/generate-example-project/server/wire"
)

func registerRootMiddleware(api huma.API) {
	// 中间件添加处
	//api.UseMiddleware()
}

// registerRootAPI 注册根级路由：http(s)://ip:port/**，不带服务名前缀，
// 供健康检查这类需要稳定短路径的探针接口使用。
func registerRootAPI(api huma.API) {
	wire.NewHealthCheckController().Register(api)
}
