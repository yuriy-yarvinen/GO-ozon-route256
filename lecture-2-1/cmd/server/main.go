package main

import (
	"context"
	"fmt"
	"log"
	"net"

	greeterv1 "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/greeter/v1"
	"google.golang.org/grpc"
)

const address = ":50051"

type greeterServer struct {
	greeterv1.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{
		Message: fmt.Sprintf("Hello, %s! from %s!", req.GetName(), req.GetGgg()),
	}, nil
}

func main() {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", address, err)
	}

	server := grpc.NewServer()
	greeterv1.RegisterGreeterServer(server, &greeterServer{})

	log.Printf("gRPC server listening on %s", address)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve gRPC: %v", err)
	}
}
