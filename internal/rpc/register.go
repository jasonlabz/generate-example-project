package rpc

import (
	"google.golang.org/grpc"

	hellov1 "github.com/jasonlabz/generate-example-project/api/proto/hello/v1"
)

// Register 注册所有 gRPC 服务实现，新增服务在此挂载。
func Register(srv *grpc.Server) {
	hellov1.RegisterHelloServiceServer(srv, NewHelloServer())
}
