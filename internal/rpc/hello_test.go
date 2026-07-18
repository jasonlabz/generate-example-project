package rpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	hellov1 "github.com/jasonlabz/generate-example-project/api/proto/hello/v1"
)

func TestSayHello(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	Register(srv)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	resp, err := hellov1.NewHelloServiceClient(conn).SayHello(context.Background(),
		&hellov1.SayHelloRequest{Name: "gopher"})
	if err != nil {
		t.Fatalf("say hello: %v", err)
	}
	if resp.GetGreeting() != "hello, gopher" {
		t.Fatalf("unexpected greeting: %s", resp.GetGreeting())
	}
}
