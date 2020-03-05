package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/serverlessresearch/srk/srkServer/srkproto"
	"google.golang.org/grpc"
)

type srkServer struct {
	srkproto.UnimplementedTestServiceServer
}

func (s *srkServer) CopyFile(context.Context, *srkproto.CopyFileArg) (*empty.Empty, error) {
	return &empty.Empty{}, srk.CopyFile("./t1", "./t2")
}

func main() {
	fmt.Println("Server starting up")
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		fmt.Println("Couldn't listen on port")
		os.Exit(1)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	srkproto.RegisterTestServiceServer(grpcServer, &srkServer{})
	fmt.Println("Server ready")
	grpcServer.Serve(listener)
}
