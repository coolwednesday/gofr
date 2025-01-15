package grpc

import (
	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/container"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HelloGoFrClient interface {
	SayHello(*gofr.Context, *HelloRequest) (*HelloResponse, error)
}

type HelloClientWrapper struct {
	client    HelloClient
	Container *container.Container
	HelloGoFrClient
}

func createGRPCConn(host string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func NewHelloGoFrClient(conn grpc.ClientConnInterface) *HelloClientWrapper {
	res := NewHelloClient(conn)
	return &HelloClientWrapper{
		client: res,
	}
}

func (h *HelloClientWrapper) SayHello(ctx *gofr.Context, req *HelloRequest) (*HelloResponse, error) {
	return h.client.SayHello(ctx.Context, req)
}
