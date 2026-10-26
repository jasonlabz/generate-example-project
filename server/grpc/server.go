package grpcserver

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/jasonlabz/generate-example-project/bootstrap"
	"github.com/jasonlabz/generate-example-project/common/resource"
)

func InitGRPCServer(c *bootstrap.Config) *grpc.Server {
	grpcCfg := c.GetServerConfig().GRPC
	recoveryOpts := []recovery.Option{recovery.WithRecoveryHandler(func(p any) error {
		resource.Logger.Errorf(context.Background(), "[grpc] recovered panic: %v\n%s", p, debug.Stack())
		return status.Error(codes.Internal, "internal server error")
	})}
	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     300 * time.Second,
			MaxConnectionAge:      1800 * time.Second,
			MaxConnectionAgeGrace: 30 * time.Second,
			Time:                  30 * time.Second,
			Timeout:               10 * time.Second,
		}),
		grpc.MaxConcurrentStreams(grpcCfg.MaxConcurrentStreams),
		grpc.ChainUnaryInterceptor(recovery.UnaryServerInterceptor(recoveryOpts...)),
		grpc.ChainStreamInterceptor(recovery.StreamServerInterceptor(recoveryOpts...)),
	)
	registerServices(srv)
	return srv
}

// registerServices 注册全部对外 gRPC 服务。
// 契约快照工具复用本函数，保证"实际注册的服务集合"只有一份定义。
func registerServices(s *grpc.Server) {

}
