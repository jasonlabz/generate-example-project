package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jasonlabz/potato/ginmetrics"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
	"github.com/jasonlabz/generate-example-project/internal/router"
)

// buildEngine 装配 gin engine：运行模式、metrics 中间件、业务路由。
func buildEngine(c *bootstrap.Config) *gin.Engine {
	mode := gin.ReleaseMode
	if c.IsDebugMode() {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)

	engine := router.InitAPIRouter()

	prometheusConf := c.GetPrometheusConfig()
	if prometheusConf.Enable {
		m := ginmetrics.GetMonitor()
		m.SetMetricPath(prometheusConf.Path)
		m.SetSlowTime(10)
		m.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})
		m.Use(engine)
	}
	return engine
}

func startHTTPServer(c *bootstrap.Config) *http.Server {
	if !c.IsHTTPEnable() {
		return nil
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", c.GetHTTPPort()),
		Handler:      buildEngine(c),
		ReadTimeout:  c.GetHTTPReadTimeout(),
		WriteTimeout: c.GetHTTPWriteTimeout(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server listen: %v", err)
		}
	}()
	return srv
}
