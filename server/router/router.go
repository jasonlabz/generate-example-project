package router

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	knife "github.com/jasonlabz/knife4go"
	"github.com/jasonlabz/potato/configx"
	"github.com/jasonlabz/potato/middleware"

	"github.com/jasonlabz/generate-example-project/server/controller"
	health_check_manager "github.com/jasonlabz/generate-example-project/server/manager/health_check"
	health_check_service "github.com/jasonlabz/generate-example-project/server/service/health_check"
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
	api := humagin.New(router, humaConfig)
	registerRootAPI(api, newHealthCheckController())

	// 对路由进行分组，处理不同的分组，根据自己的需求定义即可
	staticRouter := router.Group("/server")
	staticRouter.Static("/", "webroot")

	serverGroup := router.Group(fmt.Sprintf("/%s", serviceName))
	// Knife4go must observe the completed Huma document when it registers debug routes.
	if debug {
		if err := knife.Init(serverGroup, knife.DocumentProviderFunc(func() ([]byte, error) {
			return api.OpenAPI().Downgrade()
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

// newHealthCheckController assembles the local health-check dependency graph.
func newHealthCheckController() controller.HealthCheckController {
	probe := health_check_manager.NewLocalProbe()
	manager := health_check_manager.NewManager(probe)
	service := health_check_service.NewService(manager)

	return controller.NewHealthCheckController(service)
}

// registerRootAPI registers routes served from the root API.
func registerRootAPI(api huma.API, healthCheckController controller.HealthCheckController) {
	healthCheckController.Register(api)
}

// 注册服務路由  http://ip:port/server_name/api/**
func registerBaseAPI(router *gin.RouterGroup) {}

// 注册組路由 http://ip:port/server_name/api/v1/**
func registerV1GroupAPI(router *gin.RouterGroup) {
	// v1.RegisterSchedulerManagerGroup(router)
}
