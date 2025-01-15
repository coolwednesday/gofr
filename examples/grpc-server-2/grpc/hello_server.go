package grpc

import (
	"gofr.dev/pkg/gofr"
)

// Register the gRPC service in your app using the following code in your main.go:
//
// grpc.RegisterHelloServerWithGofr(app, &grpc.HelloGoFrServer{})
//
// HelloGoFrServer defines the gRPC server implementation.
// Customize the struct with required dependencies and fields as needed.

type HelloGoFrServer struct {
}

func (s *HelloGoFrServer) SayHello(ctx *gofr.Context) (any, error) {
	request := HelloRequest{}

	err := ctx.Bind(&request)
	if err != nil {
		return nil, err
	}

	name := request.Name
	if name == "" {
		name = "World"
	}

	conn, err := createGRPCConn("localhost:10000")
	if err != nil {
		return nil, err
	}

	service := NewHelloGoFrClient(conn)

	res, err := service.SayHello(ctx, &request)

	return res, err
}
