package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	greeterv1 "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/greeter/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const address = "localhost:50051"

func main() {
	name := "gRPC"
	ggg := "gRPC"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", address, err)
	}
	defer conn.Close()

	client := greeterv1.NewGreeterClient(conn)
	response, err := client.SayHello(ctx, &greeterv1.SayHelloRequest{Name: name, Ggg: ggg})
	if err != nil {
		log.Fatalf("failed to call SayHello: %v", err)
	}

	fmt.Println(response.GetMessage())
}
