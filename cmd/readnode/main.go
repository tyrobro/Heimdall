package main

import (
	"fmt"
	"log"
	"net"

	"heimdall/core"
	pb "heimdall/proto"

	"google.golang.org/grpc"
)

type readNodeServer struct {
	pb.UnimplementedDataServiceServer
}

func (s *readNodeServer) ReadAction(req *pb.ReadRequest, stream pb.DataService_ReadActionServer) error {
	fileName := req.GetFileName()
	targetTS := req.GetTimestamp()

	log.Printf("Gateway requested %s at TS:%d", fileName, targetTS)

	hashes, err := core.GetFileRecipe(fileName, targetTS)
	if err != nil {
		return err
	}

	for _, hash := range hashes {
		data, cacheHit, err := core.GetChunk(hash)
		if err != nil {
			return fmt.Errorf("failed to retrieve chunk %s: %w", hash, err)
		}

		if cacheHit {
			log.Printf("Cache HIT for chunk: %s", hash[:8])
		} else {
			log.Printf("Cache MISS (Loaded from disk): %s", hash[:8])
		}

		res := &pb.ReadResponse{
			ChunkData: data,
		}
		if err := stream.Send(res); err != nil {
			return fmt.Errorf("failed to send chunk stream: %w", err)
		}
	}

	log.Printf("Successfully streamed %s back to Gateway.", fileName)
	return nil
}

func main() {
	log.Println("Booting Heimdall's Read Node...")

	if err := core.InitVault(); err != nil {
		log.Fatalf("Vault failed: %v", err)
	}
	defer core.CloseVault()

	if err := core.InitCache(1000); err != nil {
		log.Fatalf("Cache failed: %v", err)
	}

	port := ":50052"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Read Node failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDataServiceServer(s, &readNodeServer{})

	log.Printf("Read Node is online and listening on port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Read Node crashed: %v", err)
	}
}
