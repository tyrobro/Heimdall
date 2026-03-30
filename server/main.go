package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"time"

	pb "heimdall/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedNodeServer
}

func (s *server) GenerateHash(ctx context.Context, req *pb.HashRequest) (*pb.HashResponse, error) {
	data := req.GetInputData()
	log.Printf("Received request to hash: %v", data)
	hasher := sha256.New()
	hasher.Write([]byte(data))
	hashString := hex.EncodeToString(hasher.Sum(nil))
	return &pb.HashResponse{
		Hash:      hashString,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

func main() {
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterNodeServer(s, &server{})

	log.Printf("Heimdall has awoken and listening to port %v", port)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
