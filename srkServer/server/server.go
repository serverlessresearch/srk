package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/serverlessresearch/srk/pkg/srkmgr"
	"github.com/serverlessresearch/srk/srkServer/srkproto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type pbByteStream interface {
	Recv() (*srkproto.ByteTransfer, error)
}

//Implements io.Reader for protobuf streams of bytes (via the ByteTransfer message type)
type pbReader struct {
	chunks  pbByteStream
	lastBuf *srkproto.ByteTransfer
	index   int64
}

func (r *pbReader) Read(p []byte) (n int, err error) {
	if r.lastBuf == nil {
		if r.lastBuf, err = r.chunks.Recv(); err != nil {
			return 0, err
		}
		r.index = 0
	}
	//We now have a valid chunks buffer and index

	n = copy(p, r.lastBuf.Chunk[r.index:])
	r.index += int64(n)

	if r.index >= int64(len(r.lastBuf.Chunk)) {
		r.lastBuf = nil
		r.index = 0
	}

	return n, nil
}

type srkServer struct {
	srkproto.UnimplementedTestServiceServer
	mgr *srkmgr.SrkManager
}

func (s *srkServer) CopyFile(context.Context, *srkproto.CopyFileArg) (*empty.Empty, error) {
	return &empty.Empty{}, srk.CopyFile("./t1", "./t2")
}

func (s *srkServer) Package(ctx context.Context, chunks srkproto.FunctionService_PackageServer) error {
	//Store the incoming tar into a temporary file, we'll unpack it and then delete it
	return errors.New("Not implemented")
}

// Creates a new srk manager (interface to SRK). Be sure to call mgr.Destroy()
// to clean up (failure to do so may require manual cleanup for open-lambda)
func getMgr() *srkmgr.SrkManager {
	mgrArgs := map[string]interface{}{}
	mgrArgs["config-file"] = "./srk.yaml"
	srkLogger := logrus.New()
	srkLogger.SetLevel(logrus.WarnLevel)
	mgrArgs["logger"] = srkLogger

	mgr, err := srkmgr.NewManager(mgrArgs)
	if err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	return mgr
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
	srkproto.RegisterTestServiceServer(grpcServer, &srkServer{mgr: getMgr()})
	fmt.Println("Server ready")
	grpcServer.Serve(listener)
}
