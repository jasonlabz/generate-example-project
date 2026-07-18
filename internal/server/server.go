package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
)

const shutdownTimeout = 5 * time.Second

// Run 按配置启动 HTTP/gRPC/pprof 服务，阻塞至 ctx 取消或收到退出信号，统一优雅退出。
func Run(ctx context.Context) {
	cfg := bootstrap.GetConfig()

	httpSrv := startHTTPServer(cfg)
	pprofSrv := startPProfServer(cfg)
	grpcSrv, grpcLis := startGRPCServer(cfg)

	if httpSrv == nil && pprofSrv == nil && grpcSrv == nil {
		log.Println("no service enabled, exiting")
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-ctx.Done():
	case <-quit:
	}
	log.Println("Shutdown Server ...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownHTTP(shutdownCtx, "pprof server", pprofSrv)
	shutdownHTTP(shutdownCtx, "http server", httpSrv)
	shutdownGRPC(shutdownCtx, grpcSrv, grpcLis)
	log.Println("Server exiting")
}

func shutdownHTTP(ctx context.Context, name string, srv *http.Server) {
	if srv == nil {
		return
	}
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("%s shutdown failed: %v", name, err)
	}
}

func shutdownGRPC(ctx context.Context, srv *grpc.Server, lis net.Listener) {
	if srv == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		srv.Stop()
	}
	if lis != nil {
		_ = lis.Close()
	}
}
