package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	deprecatedexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/deprecated"
	enumexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/enum"
	googlecommontypes "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/googlecommontypes"
	mapsexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/maps"
	oneofexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/oneof"
	optionalexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/optional"
	scalarvaluetypes "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/scalarvaluetypes"
	examplesv1 "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/v1"
	wellknowntypes "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/wellknowntypes"
	greeterv1 "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/greeter/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const defaultAddress = ":50051"

type greeterServer struct {
	greeterv1.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{
		Message: fmt.Sprintf("Hello, %s! from %s!", req.GetName(), req.GetGgg()),
	}, nil
}

type examplesServer struct {
	examplesv1.UnimplementedExamplesServer
}

func (*examplesServer) ScalarValueTypes(_ context.Context, req *scalarvaluetypes.ScalarValueTypesExample) (*scalarvaluetypes.ScalarValueTypesExample, error) {
	return req, nil
}

func (*examplesServer) Enum(_ context.Context, req *enumexample.EnumExample) (*enumexample.EnumExample, error) {
	return req, nil
}

func (*examplesServer) Maps(_ context.Context, req *mapsexample.MapsExample) (*mapsexample.MapsExample, error) {
	return req, nil
}

func (*examplesServer) OneOf(_ context.Context, req *oneofexample.OneOfExample) (*oneofexample.OneOfExample, error) {
	return req, nil
}

func (*examplesServer) Optional(_ context.Context, req *optionalexample.OptionalExample) (*optionalexample.OptionalExample, error) {
	return req, nil
}

func (*examplesServer) Deprecated(_ context.Context, req *deprecatedexample.DeprecatedExample) (*deprecatedexample.DeprecatedExample, error) {
	return req, nil
}

func (*examplesServer) GoogleWellKnownTypes(_ context.Context, req *wellknowntypes.GoogleWellKnownTypesExample) (*wellknowntypes.GoogleWellKnownTypesExample, error) {
	return req, nil
}

func (*examplesServer) GoogleCommonTypes(_ context.Context, req *googlecommontypes.GoogleCommonTypesExample) (*googlecommontypes.GoogleCommonTypesExample, error) {
	return req, nil
}

type deprecatedServer struct {
	deprecatedexample.UnimplementedDeprecatedServiceServer
}

func (*deprecatedServer) OldMethod(_ context.Context, req *deprecatedexample.DeprecatedExample) (*deprecatedexample.DeprecatedExample, error) {
	return req, nil
}

type healthCheckServer struct {
	wellknowntypes.UnimplementedHealthCheckServer
}

func (*healthCheckServer) Check(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func main() {
	address := os.Getenv("GRPC_ADDR")
	if address == "" {
		address = defaultAddress
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", address, err)
	}

	server := grpc.NewServer()
	greeterv1.RegisterGreeterServer(server, &greeterServer{})
	examplesv1.RegisterExamplesServer(server, &examplesServer{})
	deprecatedexample.RegisterDeprecatedServiceServer(server, &deprecatedServer{})
	wellknowntypes.RegisterHealthCheckServer(server, &healthCheckServer{})

	log.Printf("gRPC server listening on %s", address)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve gRPC: %v", err)
	}
}
