package main

import (
	"context"
	"log"
	"time"

	pb "heimdall/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost: 50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	defer conn.Close()

	c := pb.NewNodeClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	message := "Hello!"
	log.Printf("Sending message: %s", message)
	r, err := c.GenerateHash(ctx, &pb.HashRequest{InputData: message})
	if err != nil {
		log.Fatalf("RPC failed: %v", err)
	}

	log.Printf("Success")
	log.Printf("Hash Received: %s", r.GetHash())
	log.Printf("Server Time: %s", r.GetTimestamp())
}
