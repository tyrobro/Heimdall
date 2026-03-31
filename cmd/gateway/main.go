package main

import (
	"context"
	"io"
	"log"
	"net"
	"time"

	"heimdall/core"
	pb "heimdall/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type gatewayServer struct {
	pb.UnimplementedDataServiceServer
}

func (s *gatewayServer) WriteAction(stream pb.DataService_WriteActionServer) error {
	opsProcessed.WithLabelValues("write").Inc()

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := pb.NewDataServiceClient(conn)

	backendStream, err := client.WriteAction(context.Background())
	if err != nil {
		return err
	}

	packet1, err := stream.Recv()
	if err != nil {
		return err
	}
	fileName := "unknown"
	if meta := packet1.GetMetadata(); meta != nil {
		fileName = meta.FileName
	}

	packet2, err := stream.Recv()
	if err != nil {
		return err
	}

	chunk := packet2.GetChunkData()
	bytesIngested.Add(float64(len(chunk)))

	var magicBytes []byte
	if len(chunk) > 512 {
		magicBytes = chunk[:512]
	} else {
		magicBytes = chunk
	}

	start := time.Now()
	prediction := PredictWorkload(fileName, magicBytes)
	inferenceLatency.Observe(time.Since(start).Seconds())
	log.Printf("🧠 ML Brain predicted [%s] as: %s", fileName, prediction)

	if err := backendStream.Send(packet1); err != nil {
		return err
	}
	if err := backendStream.Send(packet2); err != nil {
		return err
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			res, err := backendStream.CloseAndRecv()
			if err != nil {
				return err
			}
			LogEvent(fileName, "WRITE")
			return stream.SendAndClose(res)
		}
		if err != nil {
			return err
		}

		if data := req.GetChunkData(); data != nil {
			bytesIngested.Add(float64(len(data)))
		}

		if err := backendStream.Send(req); err != nil {
			return err
		}
	}
}

func (s *gatewayServer) ReadAction(req *pb.ReadRequest, stream pb.DataService_ReadActionServer) error {
	opsProcessed.WithLabelValues("read").Inc()

	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := pb.NewDataServiceClient(conn)

	backendStream, err := client.ReadAction(context.Background(), req)
	if err != nil {
		return err
	}

	fileName := req.GetFileName()

	for {
		res, err := backendStream.Recv()
		if err == io.EOF {
			LogEvent(fileName, "READ")
			log.Printf("Logged READ event for: %s", fileName)
			return nil
		}
		if err != nil {
			return err
		}

		if err := stream.Send(res); err != nil {
			return err
		}
	}
}

func (s *gatewayServer) ListFiles(ctx context.Context, req *pb.EmptyRequest) (*pb.ListResponse, error) {
	files, err := core.GetAllFiles()
	if err != nil {
		return nil, err
	}
	return &pb.ListResponse{Files: files}, nil
}

func main() {
	InitTelemetry()
	defer CloseTelemetry()

	StartMetricsServer()
	log.Println("Metrics exporter online at http://localhost:2112/metrics")

	port := ":50050"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Gateway failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDataServiceServer(s, &gatewayServer{})

	log.Printf("Heimdall Gateway routing traffic & logging telemetry on port %s...", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Gateway failed to serve: %v", err)
	}
}
