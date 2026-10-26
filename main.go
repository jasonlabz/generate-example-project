package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jasonlabz/potato/ginmetrics"
	"google.golang.org/grpc"

	"github.com/jasonlabz/generate-example-project/bootstrap"
	grpcserver "github.com/jasonlabz/generate-example-project/server/grpc"
	"github.com/jasonlabz/generate-example-project/server/router"
)

func main() {
	// context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// bootstrap init
	bootstrap.MustInit(ctx)

	// gin mode
	serverMode := gin.ReleaseMode
	serverConfig := bootstrap.GetConfig()
	if serverConfig.IsDebugMode() {
		serverMode = gin.DebugMode
	}
	gin.SetMode(serverMode)

	r, err := router.InitApiRouter()
	if err != nil {
		log.Fatalf("initialize API router: %v", err)
	}

	prometheusConf := serverConfig.GetPrometheusConfig()
	if prometheusConf.Enable {
		// get global Monitor object
		m := ginmetrics.GetMonitor()

		// +optional set metric path, default /debug/metrics
		m.SetMetricPath(prometheusConf.Path)
		// +optional set slow time, default 5s
		m.SetSlowTime(10)
		// +optional set request duration, default {0.1, 0.3, 1.2, 5, 10}
		// used to p95, p99
		m.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})

		// set middleware for gin
		m.Use(r)
	}

	// receive quit signal, ready to exit
	quit := make(chan os.Signal, 1)
	pprofSrv := startPProfServer(r, serverConfig, quit)
	grpcSrv, grpcLis := startGRPCServer(serverConfig, quit)

	// start program
	srv := startHTTPServer(r, serverConfig, quit)

	if srv == nil && pprofSrv == nil && grpcSrv == nil {
		log.Println("no service enabled, exiting")
		return
	}

	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-quit
	log.Println("Shutdown Server ...")
	cancel()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if pprofSrv != nil {
		if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
			log.Printf("pprof server shutdown failed: %v", err)
		}
	}
	if srv != nil {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("http server shutdown failed: %v", err)
		}
	}
	if grpcSrv != nil {
		stopDone := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(stopDone)
		}()
		select {
		case <-stopDone:
		case <-shutdownCtx.Done():
			grpcSrv.Stop()
		}
	}
	if grpcLis != nil {
		_ = grpcLis.Close()
	}
	log.Println("Server exiting ...")
}

// startHTTPServer 自定义http配置
func startHTTPServer(router *gin.Engine, c *bootstrap.Config, quit chan os.Signal) *http.Server {
	if !c.IsHTTPEnable() {
		return nil
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", c.GetHTTPPort()),
		Handler:      router,
		ReadTimeout:  c.GetHTTPReadTimeout(),
		WriteTimeout: c.GetHTTPWriteTimeout(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			notifyInterrupt(quit)
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return srv
}

func startPProfServer(router *gin.Engine, c *bootstrap.Config, quit chan os.Signal) *http.Server {
	pprofConf := c.GetPProfConfig()
	if !pprofConf.Enable {
		return nil
	}

	router.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
	srv := &http.Server{Addr: fmt.Sprintf(":%d", pprofConf.Port), Handler: nil}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			notifyInterrupt(quit)
			log.Fatalf("pprof server failed: %v", err)
		}
	}()
	return srv
}

func startGRPCServer(c *bootstrap.Config, quit chan os.Signal) (*grpc.Server, net.Listener) {
	if !c.IsGRPCEnable() {
		return nil, nil
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.GetGRPCPort()))
	if err != nil {
		notifyInterrupt(quit)
		log.Fatalf("grpc listen failed: %v", err)
	}

	srv := grpcserver.InitGRPCServer(c)
	go func() {
		if err = srv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) && !errors.Is(err, net.ErrClosed) {
			notifyInterrupt(quit)
			log.Fatalf("grpc server failed: %v", err)
		}
	}()

	return srv, lis
}

func notifyInterrupt(quit chan os.Signal) {
	select {
	case quit <- os.Interrupt:
	default:
	}
}
