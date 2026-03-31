package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "heimdall/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("=== Initiating Cluster Integration Test ===")

	conn, err := grpc.NewClient("localhost:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Gateway: %v", err)
	}
	defer conn.Close()

	client := pb.NewDataServiceClient(conn)
	fileName := "network_test_payload.txt"
	testData := []byte("This is a highly classified payload transmitted over the Heimdall network stream.\n")

	var fullPayload bytes.Buffer
	for i := 0; i < 1000; i++ {
		fullPayload.Write(testData)
	}

	fmt.Println("\n[Phase 1] Testing Gateway -> Write Node Routing...")

	writeStream, err := client.WriteAction(context.Background())
	if err != nil {
		log.Fatalf("Write stream failed to open: %v", err)
	}

	err = writeStream.Send(&pb.WriteRequest{
		Request: &pb.WriteRequest_Metadata{
			Metadata: &pb.FileMetadata{FileName: fileName},
		},
	})
	if err != nil {
		log.Fatalf("Failed to send metadata: %v", err)
	}

	chunkSize := 4096
	payloadBytes := fullPayload.Bytes()

	for i := 0; i < len(payloadBytes); i += chunkSize {
		end := i + chunkSize
		if end > len(payloadBytes) {
			end = len(payloadBytes)
		}

		err = writeStream.Send(&pb.WriteRequest{
			Request: &pb.WriteRequest_ChunkData{
				ChunkData: payloadBytes[i:end],
			},
		})
		if err != nil {
			log.Fatalf("Failed to send chunk: %v", err)
		}
	}

	writeRes, err := writeStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Printf("Upload Success! Saved at Timestamp: %d\n", writeRes.Timestamp)

	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n[Phase 2] Testing Gateway -> Read Node Routing...")

	readStream, err := client.ReadAction(context.Background(), &pb.ReadRequest{
		FileName:  fileName,
		Timestamp: writeRes.Timestamp,
	})
	if err != nil {
		log.Fatalf("Read stream failed to open: %v", err)
	}

	var downloadedData bytes.Buffer
	for {
		res, err := readStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Download stream failed: %v", err)
		}
		downloadedData.Write(res.ChunkData)
	}

	fmt.Printf("Download Success! Retrieved %d bytes from the cluster.\n", downloadedData.Len())

	if bytes.Equal(fullPayload.Bytes(), downloadedData.Bytes()) {
		fmt.Println("\nINTEGRATION TEST PASSED: Byte-for-byte match verified across the network!")
	} else {
		fmt.Println("\nINTEGRATION TEST FAILED: Data corruption detected in transit.")
	}

	os.Remove("temp_" + fileName)
}
