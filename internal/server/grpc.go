package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
	"github.com/jasonlabz/generate-example-project/internal/rpc"
)

func startGRPCServer(c *bootstrap.Config) (*grpc.Server, net.Listener) {
	if !c.IsGRPCEnable() {
		return nil, nil
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.GetGRPCPort()))
	if err != nil {
		log.Fatalf("grpc listen failed: %v", err)
	}

	grpcCfg := c.GetServerConfig().GRPC
	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     300 * time.Second,
			MaxConnectionAge:      1800 * time.Second,
			MaxConnectionAgeGrace: 30 * time.Second,
			Time:                  30 * time.Second,
			Timeout:               10 * time.Second,
		}),
		grpc.MaxConcurrentStreams(grpcCfg.MaxConcurrentStreams),
	)

	rpc.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, net.ErrClosed) {
			log.Printf("grpc server failed: %v", serveErr)
		}
	}()
	return srv, lis
}
