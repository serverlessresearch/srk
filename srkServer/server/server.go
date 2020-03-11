package main

import (
	"context"
	"fmt"
	"io"
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
	return &empty.Empty{}, srk.CopyFile("./testData/t1", "./testData/testOutput/t2")
}

// Package implements a gRPC wrapper around srk.FunctionService.Package().
// chunks represents a stream of bytes representing a tar file containing
// everything that should be included in the function. This tar should extract
// to a single top-level folder. The name of this folder will be used as the
// name of the function.
func (s *srkServer) Package(chunks srkproto.FunctionService_PackageServer) error {
	funcReader := &pbReader{chunks: chunks}

	//XXX temporary test
	fmt.Println("Receiving file")
	dst, err := os.OpenFile("testData/testOutput/t1", os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	defer dst.Close()
	fmt.Println("Opened file")
	io.Copy(dst, funcReader)
	fmt.Println("Transfer Complete")

	return nil
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
