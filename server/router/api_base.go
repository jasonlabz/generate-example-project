package router

import "github.com/danielgtaylor/huma/v2"

func registerBaseMiddleware(api huma.API) {
	// 中间件添加处
	//api.UseMiddleware()
}

// registerBaseAPI 注册服务基础路由：http(s)://ip:port/<服务名>/**
// 参数 middleware 为 huma 组中间件（huma.Middlewares），需要时传入
// huma.Group.UseMiddleware 挂载。
func registerBaseAPI(api huma.API) {
}
