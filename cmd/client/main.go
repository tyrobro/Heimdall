package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	pb "heimdall/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	conn, err := grpc.NewClient("localhost:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Heimdall Gateway: %v", err)
	}
	defer conn.Close()
	client := pb.NewDataServiceClient(conn)

	switch command {
	case "upload":
		handleUpload(client)
	case "download":
		handleDownload(client)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Heimdall Storage CLI")
	fmt.Println("Usage:")
	fmt.Println("  upload <file_path>")
	fmt.Println("  download <file_name> <timestamp> <output_path>")
}

func handleUpload(client pb.DataServiceClient) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: upload <file_path>")
		os.Exit(1)
	}

	filePath := os.Args[2]
	fileName := filepath.Base(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	stream, err := client.WriteAction(context.Background())
	if err != nil {
		log.Fatalf("Failed to open upload stream: %v", err)
	}

	err = stream.Send(&pb.WriteRequest{
		Request: &pb.WriteRequest_Metadata{
			Metadata: &pb.FileMetadata{FileName: fileName},
		},
	})
	if err != nil {
		log.Fatalf("Failed to send metadata: %v", err)
	}

	buf := make([]byte, 64*1024)
	fmt.Printf("Uploading %s...\n", fileName)

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}

		err = stream.Send(&pb.WriteRequest{
			Request: &pb.WriteRequest_ChunkData{
				ChunkData: buf[:n],
			},
		})
		if err != nil {
			log.Fatalf("Failed to send chunk: %v", err)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Printf("Success! %s\n", res.Message)
	fmt.Printf("Version Timestamp: %d\n", res.Timestamp)
}

func handleDownload(client pb.DataServiceClient) {
	if len(os.Args) < 5 {
		fmt.Println("Usage: download <file_name> <timestamp> <output_path>")
		os.Exit(1)
	}

	fileName := os.Args[2]
	timestamp, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil {
		log.Fatalf("Invalid timestamp: %v", err)
	}
	outputPath := os.Args[4]

	stream, err := client.ReadAction(context.Background(), &pb.ReadRequest{
		FileName:  fileName,
		Timestamp: timestamp,
	})
	if err != nil {
		log.Fatalf("Failed to open download stream: %v", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	fmt.Printf("Downloading %s (Version %d)...\n", fileName, timestamp)

	var totalBytes int
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Download stream error: %v", err)
		}

		n, err := outFile.Write(res.ChunkData)
		if err != nil {
			log.Fatalf("Failed to write to disk: %v", err)
		}
		totalBytes += n
	}

	fmt.Printf("Success! Downloaded %d bytes to %s\n", totalBytes, outputPath)
}
