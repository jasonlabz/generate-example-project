package router

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	knife "github.com/jasonlabz/knife4go"
	"github.com/jasonlabz/potato/configx"
	_middleware "github.com/jasonlabz/potato/middleware"

	"github.com/jasonlabz/generate-example-project/server/wire/health_check"
)

// InitApiRouter creates the API router from the global server configuration.
func InitApiRouter() (*gin.Engine, error) {
	router := gin.New()

	serverConfig := configx.GetConfig()

	//modules := appwire.Modules()

	// Engine scope: recovery protects every route, including non-Huma assets.
	rootMiddleware(router,
		_middleware.RecoveryLog(true),
		_middleware.SetContext(),
		_middleware.RequestMiddleware(),
	)

	serverGroup := router.Group(fmt.Sprintf("/%s", serverConfig.GetName()))
	humaConfig := huma.DefaultConfig(serverConfig.GetName(), "v1")
	humaConfig.DocsPath = ""
	humaConfig.OpenAPIPath = ""
	humaConfig.SchemasPath = ""
	// Disable schema links to preserve the existing HTTP response contract.
	humaConfig.CreateHooks = nil

	// Huma owns the application route hierarchy. Groups compose prefixes and
	// OpenAPI metadata while retaining the same underlying Gin adapter.
	serverAPI := humagin.NewWithGroup(router, serverGroup, humaConfig)
	//registerBaseAPI(serverAPI, modules...)

	// Knife4go is a Gin-based documentation asset, so it is the only route
	// infrastructure that still receives a Gin group.
	if serverConfig.IsDebugMode() {
		if err := knife.Init(serverGroup, knife.DocumentProviderFunc(func() ([]byte, error) {
			return serverAPI.OpenAPI().Downgrade()
		})); err != nil {
			return nil, fmt.Errorf("initialize knife4go documentation: %w", err)
		}
	}
	apiGroup := huma.NewGroup(serverAPI, "/api")

	v1Group := huma.NewGroup(apiGroup, "/v1")
	registerV1GroupAPI(v1Group)

	// may be change to /v2 api group
	//v2Group := huma.NewGroup(apiGroup, "/v2")
	//registerV1GroupAPI(v2Group, modules...)
	return router, nil
}

func rootMiddleware(r *gin.Engine, middlewares ...gin.HandlerFunc) {
	r.Use(middlewares...)
}

// registerRootAPI registers all module routes served from the server API.\
func registerRootAPI(api huma.API, middleware ...huma.Middlewares) {
	health_check.NewController().Register(api)
}

// 注册服務路由  http://ip:port/server_name/**
func registerBaseAPI(api huma.API, middleware ...huma.Middlewares) {

}

// 注册組路由 http://ip:port/server_name/api/v1/**
func registerV1GroupAPI(api huma.API, middleware ...huma.Middlewares) {

}
