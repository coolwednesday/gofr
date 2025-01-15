package main

import (
	"gofr.dev/examples/grpc-server-2/grpc"
	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.New()

	grpc.RegisterHelloServerWithGofr(app, &grpc.HelloGoFrServer{})

	app.Run()
}
