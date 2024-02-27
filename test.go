package main

import (
	"context"
	"fmt"

	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func testGRPC() {
	conn, err := grpc.Dial("localhost:8079", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("Connected to Gateway.\n")

	client := gateway.NewClusterGatewayClient(conn)

	fmt.Printf("Created new ClusterGatewayClient.\n")

	resp, err := client.ID(context.Background(), &gateway.Void{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %v\n", resp)
}

func main() {
	// testWalkDir()
	testGRPC()
}
