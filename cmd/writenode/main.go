package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"heimdall/core"
	pb "heimdall/proto"

	"google.golang.org/grpc"
)

type writeNodeServer struct {
	pb.UnimplementedDataServiceServer
}

func (s *writeNodeServer) WriteAction(stream pb.DataService_WriteActionServer) error {
	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive initial packet: %w", err)
	}

	meta := req.GetMetadata()
	if meta == nil {
		return fmt.Errorf("first packet must contain FileMetadata")
	}
	fileName := meta.FileName

	tempPath := "temp_" + fileName
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	log.Printf("Receiving stream for: %s", fileName)
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			tempFile.Close()
			return err
		}

		chunk := req.GetChunkData()
		if chunk != nil {
			tempFile.Write(chunk)
		}
	}
	tempFile.Close()
	ts := core.Oracle.GetNextTimestamp()

	err = core.AsyncSave(tempPath, fileName, ts)
	if err != nil {
		return stream.SendAndClose(&pb.WriteResponse{
			Success: false,
			Message: err.Error(),
		})
	}
	log.Printf("Successfully ingested %s to WAL at TS:%d", fileName, ts)

	return stream.SendAndClose(&pb.WriteResponse{
		Success:   true,
		Timestamp: ts,
		Message:   "File successfully ingested into Heimdall Storage Cluster",
	})
}

func main() {
	log.Println("Booting Heimdall...")
	if err := core.InitVault(); err != nil {
		log.Fatalf("Vault failed: %v", err)
	}
	defer core.CloseVault()

	core.InitTSO()

	if err := core.InitWAL(); err != nil {
		log.Fatalf("WAL failed: %v", err)
	}
	defer core.CloseWAL()

	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Write Node failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDataServiceServer(s, &writeNodeServer{})

	log.Printf("Write node is online and listening to port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Write Node crashed: %v", err)
	}
}
