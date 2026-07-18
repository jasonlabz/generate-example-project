package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	// pprof 性能分析端点注册到 http.DefaultServeMux
	_ "net/http/pprof"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
)

func startPProfServer(c *bootstrap.Config) *http.Server {
	pprofConf := c.GetPProfConfig()
	if !pprofConf.Enable {
		return nil
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", pprofConf.Port), Handler: nil}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()
	return srv
}
