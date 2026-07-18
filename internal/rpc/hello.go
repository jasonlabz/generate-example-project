package rpc

import (
	"context"
	"fmt"

	hellov1 "github.com/jasonlabz/generate-example-project/api/proto/hello/v1"
)

// HelloServer HelloService 示例实现：rpc 层是薄适配层，业务逻辑应调用 internal/service。
type HelloServer struct {
	hellov1.UnimplementedHelloServiceServer
}

func NewHelloServer() *HelloServer { return &HelloServer{} }

func (s *HelloServer) SayHello(_ context.Context, req *hellov1.SayHelloRequest) (*hellov1.SayHelloResponse, error) {
	return &hellov1.SayHelloResponse{Greeting: fmt.Sprintf("hello, %s", req.GetName())}, nil
}
