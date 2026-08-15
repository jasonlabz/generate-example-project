package router

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	knife "github.com/jasonlabz/knife4go"
	"github.com/jasonlabz/potato/configx"
	"github.com/jasonlabz/potato/middleware"

	healthcheckwire "github.com/jasonlabz/generate-example-project/server/wire/health_check"
)

// InitApiRouter creates the API router from the global server configuration.
func InitApiRouter() (*gin.Engine, error) {
	serverConfig := configx.GetConfig()
	return newAPIRouter(serverConfig.GetName(), serverConfig.IsDebugMode())
}

// newAPIRouter creates an API router from deterministic inputs.
func newAPIRouter(serviceName string, debug bool) (*gin.Engine, error) {
	router := gin.New()

	// 全局中间件，查看定义的中间价在middlewares文件夹中
	rootMiddleware(router)

	humaConfig := huma.DefaultConfig(serviceName, "v1")
	humaConfig.DocsPath = ""
	humaConfig.OpenAPIPath = ""
	humaConfig.SchemasPath = ""
	// Disable schema links to preserve the existing HTTP response contract.
	humaConfig.CreateHooks = nil
	routerApi := humagin.New(router, humaConfig)
	registerRootAPI(routerApi)

	serverGroup := router.Group(fmt.Sprintf("/%s", serviceName))
	// Knife4go must observe the completed Huma document when it registers debug routes.
	if debug {
		if err := knife.Init(serverGroup, knife.DocumentProviderFunc(func() ([]byte, error) {
			return routerApi.OpenAPI().Downgrade()
		})); err != nil {
			return nil, fmt.Errorf("initialize knife4go documentation: %w", err)
		}
	}

	// base api
	registerBaseAPI(serverGroup)

	apiGroup := serverGroup.Group("/api")

	// 中间件拦截器
	groupMiddleware(apiGroup,
		middleware.RecoveryLog(true), middleware.SetContext(), middleware.RequestMiddleware())

	// v1 group api
	v1Group := apiGroup.Group("/v1")
	registerV1GroupAPI(v1Group)

	return router, nil
}

func rootMiddleware(r *gin.Engine, middlewares ...gin.HandlerFunc) {
	r.Use(middlewares...)
}

func groupMiddleware(g *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	g.Use(middlewares...)
}

// registerRootAPI registers routes served from the root API.
func registerRootAPI(api huma.API) {
	healthCheckController := healthcheckwire.NewController()
	healthCheckController.Register(api)
}

// 注册服務路由  http://ip:port/server_name/api/**
func registerBaseAPI(router *gin.RouterGroup) {}

// 注册組路由 http://ip:port/server_name/api/v1/**
func registerV1GroupAPI(router *gin.RouterGroup) {
	// v1.RegisterSchedulerManagerGroup(router)
}
