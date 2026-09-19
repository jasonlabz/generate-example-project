package router

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/jasonlabz/generate-example-project/server/wire"
)

func registerV1GroupMiddleware(api huma.API) {
	// 中间件添加处
	//api.UseMiddleware()
}

// registerV1GroupAPI 注册版本化业务路由：http(s)://ip:port/<服务名>/api/v1/**
// 业务模块的控制器（实现 Register(api huma.API) 接口）在此逐个注册。
// user 模块是新增业务的参考样例，覆盖了路径参数、查询筛选、分页、请求体、
// 文件流与错误码的完整写法。
func registerV1GroupAPI(api huma.API) {
	wire.NewUserController().Register(api)
}
