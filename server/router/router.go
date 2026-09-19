// Package router 组装应用的全部 HTTP 路由。
//
// 路由架构（huma 组层级）：
//
//	gin.Engine（底层，承载 gin 中间件与非 huma 资产）
//	└── rooterAPI（huma API，根级；health-check 等根路由可注册在此）
//	    └── serverAPI（huma 组，前缀 /<服务名>；业务路由统一挂在此组下）
//	        ├── 基础路由：/{服务名}/health-check 等
//	        └── apiGroup（huma 组，前缀 /api）
//	            └── v1Group（huma 组，前缀 /v1）→ /{服务名}/api/v1/**
//
// huma 组（huma.NewGroup）同时作用于「路由前缀」与「OpenAPI 文档 paths 前缀」：
// 组前缀会写入生成的 OpenAPI 文档，前端展示与调试请求都使用文档中的完整路径。
package router

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	knife "github.com/jasonlabz/knife4go"
	_middleware "github.com/jasonlabz/potato/middleware"

	"github.com/jasonlabz/generate-example-project/bootstrap"
	"github.com/jasonlabz/generate-example-project/common/humax"
	"github.com/jasonlabz/generate-example-project/server/middleware"
)

// InitApiRouter 根据全局配置组装 API 路由，返回 gin.Engine。
//
// 架构说明：
//   - gin 负责底层 HTTP 与 gin 中间件（恢复、上下文、请求日志）；
//   - huma 通过 humagin 适配器挂在 gin 之上，业务路由全部用 huma 注册；
//   - knife4go 负责 Knife4j 文档 UI 与 OpenAPI 文档端点（仅调试模式注册）。
func InitApiRouter() (*gin.Engine, error) {
	router := gin.New()

	serverConfig := bootstrap.GetConfig()

	// Engine 级中间件：RecoveryLog 保护包括非 huma 资产在内的全部路由。
	// 注意：gin 中间件作用于整个 gin.Engine；huma 中间件（huma.Middlewares）
	// 则通过 huma.Group.UseMiddleware 按组挂载（见 registerBaseAPI 签名）。
	router.Use(_middleware.RecoveryLog(true))

	// huma 配置：huma.DefaultConfig(serviceName, "v1") 已设置
	// Info.Title（服务名）与 Info.Version（"v1"），以下按需补充。
	humaConfig := huma.DefaultConfig(serverConfig.GetName(), "v1")

	// —— 服务信息（显示在接口文档页面头部，对应 swag 的 @title/@version/@description）——
	humaConfig.Info.Title = "generate-example-project API"
	humaConfig.Info.Description = "基于 gin + Huma v2 的分层 API 示例项目，覆盖路由、分层、文档、错误处理与工具链约定。"
	humaConfig.Info.Contact = &huma.Contact{
		Name:  "generate-example-project",
		URL:   "https://github.com/jasonlabz/generate-example-project",
		Email: "example@example.com",
	}

	// —— 文档端点：关闭 huma 自带的文档 UI 与 OpenAPI 端点 ——
	// （DocsPath/OpenAPIPath/SchemasPath 置空），文档 UI 统一交给 knife4go；
	// CreateHooks 置空以保留既有 HTTP 响应契约（关闭 schema 链接等钩子）。
	humaConfig.DocsPath = ""
	humaConfig.OpenAPIPath = ""
	humaConfig.SchemasPath = ""
	humaConfig.CreateHooks = nil
	// 所有 API 响应均回退至 JSON，避免协商失败返回非约定的 406。
	humaConfig.NoFormatFallback = false
	// 字段默认可选，与旧 Gin binding:"required" 语义对齐；必填字段显式用 required:"true" 标记。
	humaConfig.FieldsOptionalByDefault = true

	// —— 请求约束 ——
	// 请求体大小上限与读取超时在 Operation 或 huma.Config 层面按需设置；
	// 单接口级别可参考 controller 中 Operation.MaxBodyBytes 的注释示例。

	// 在创建 Huma API 前统一安装错误详情策略和校验错误工厂，避免默认 RFC7807 响应泄漏到调用方。
	humax.ConfigureErrorDetails(gin.IsDebugging())
	humax.ConfigureHumaErrorFactory("v1")

	// 错误链上报：无论响应是否脱敏，服务端日志都保留完整错误链。
	// 生产环境（debug=false）响应里没有 err_trace，这里是唯一的排查依据。
	humax.SetErrorReporter(middleware.LogErrorChain)

	// humagin.New 把 gin.Engine 适配为 huma.API：huma 路由注册在 gin 之上，
	// 两者共享同一 HTTP 服务。
	rooterAPI := humagin.New(router, humaConfig)
	registerRootAPI(rooterAPI)

	// serverAPI 是全部业务路由的根组，前缀 /<服务名>：
	// huma.NewGroup 的组前缀会同时写入路由与 OpenAPI 文档 paths。
	serverAPI := huma.NewGroup(rooterAPI, "/"+serverConfig.GetName())
	registerBaseMiddleware(serverAPI)
	registerBaseAPI(serverAPI)

	// 版本化 API 组：/api/v1 前缀，业务模块（如用户、任务等）在此注册。
	apiGroup := huma.NewGroup(serverAPI, "/api")
	registerBaseMiddleware(apiGroup)
	// 中间件：RecoveryLog 保护包括/api 下的全部路由。
	apiGroup.UseMiddleware(
		middleware.HumaRecoveryLog(true),
		middleware.SetHumaContext(),
		middleware.HumaRequestMiddleware(),
	)

	v1Group := huma.NewGroup(apiGroup, "/v1")
	registerV1GroupMiddleware(v1Group)
	registerV1GroupAPI(v1Group)

	// 新增版本：创建 /v2 组并注册对应模块（示例，按需启用）。
	//v2Group := huma.NewGroup(apiGroup, "/v2")
	//registerV1GroupAPI(v2Group, modules...)

	// 文档注册（仅调试模式）：把 serverAPI 生成的 OpenAPI 3.0 文档透传给
	// knife4go。展示与调试请求直接使用文档中的完整路径，必须放在所有路由注册之后，保证文档包含完整路由。
	// 接口文档地址：http(s)://ip:port/<服务名>/doc.html
	if gin.IsDebugging() {
		openAPIDocument, err := serverAPI.OpenAPI().Downgrade()
		if err != nil {
			return nil, fmt.Errorf("downgrade huma OpenAPI document: %w", err)
		}
		if err = knife.InitHumaKnife(serverAPI,
			knife.Doc(string(openAPIDocument))); err != nil {
			return nil, fmt.Errorf("initialize knife4go documentation: %w", err)
		}
	}
	return router, nil
}
