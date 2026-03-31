package main

import (
	"context"
	"io"
	"log"
	"net"

	pb "heimdall/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type gatewayServer struct {
	pb.UnimplementedDataServiceServer
}

func (s *gatewayServer) WriteAction(stream pb.DataService_WriteActionServer) error {
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

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			res, err := backendStream.CloseAndRecv()
			if err != nil {
				return err
			}
			return stream.SendAndClose(res)
		}
		if err != nil {
			return err
		}

		if err := backendStream.Send(req); err != nil {
			return err
		}
	}
}

func (s *gatewayServer) ReadAction(req *pb.ReadRequest, stream pb.DataService_ReadActionServer) error {
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
	for {
		res, err := backendStream.Recv()
		if err == io.EOF {
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

func main() {
	port := ":50050"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Gateway failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDataServiceServer(s, &gatewayServer{})

	log.Printf("Heimdall Gateway is routing traffic on post %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Gateway failed to serve: %v", err)
	}
}
