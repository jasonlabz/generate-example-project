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
	serverConfig := configx.GetConfig()
	return newAPIRouter(serverConfig.GetName(), serverConfig.IsDebugMode())
}

// newAPIRouter builds the API router from deterministic inputs so integration
// tests can construct it without reading global configuration.
func newAPIRouter(serviceName string, debug bool) (*gin.Engine, error) {
	router := gin.New()

	// Engine scope: recovery protects every route, including non-Huma assets.
	rootMiddleware(router,
		_middleware.RecoveryLog(true),
		_middleware.SetContext(),
		_middleware.RequestMiddleware(),
	)

	humaConfig := huma.DefaultConfig(serviceName, "v1")
	humaConfig.DocsPath = ""
	humaConfig.OpenAPIPath = ""
	humaConfig.SchemasPath = ""
	// Disable schema links to preserve the existing HTTP response contract.
	humaConfig.CreateHooks = nil

	// Root API serves operations outside the /{service} prefix, e.g. /health-check.
	api := humagin.New(router, humaConfig)
	registerRootAPI(api)

	// Knife4go is a Gin-based documentation asset served below /{service}.
	serverGroup := router.Group(fmt.Sprintf("/%s", serviceName))
	if debug {
		if err := knife.Init(serverGroup, knife.DocumentProviderFunc(func() ([]byte, error) {
			return api.OpenAPI().Downgrade()
		})); err != nil {
			return nil, fmt.Errorf("initialize knife4go documentation: %w", err)
		}
	}

	// Versioned group below /{service}/api/v1 for future modules.
	registerV1GroupAPI(serverGroup.Group("/api/v1"))

	return router, nil
}

func rootMiddleware(r *gin.Engine, middlewares ...gin.HandlerFunc) {
	r.Use(middlewares...)
}

// registerRootAPI registers operations served at the engine root.
func registerRootAPI(api huma.API) {
	health_check.NewController().Register(api)
}

// registerV1GroupAPI registers operations below /{service}/api/v1.
func registerV1GroupAPI(_ *gin.RouterGroup) {
	// v1 modules register here, e.g. user.RegisterV1(group).
}
