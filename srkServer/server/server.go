package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/serverlessresearch/srk/pkg/srkmgr"
	"github.com/serverlessresearch/srk/srkServer/srkproto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

// Represents a function registration session so that we can track matching
// RegisterFunc() and UploadFunc() calls.
// type registerSession struct {
// 	client net.Addr
// 	fname  string
// }

type srkServer struct {
	srkproto.UnimplementedFunctionServiceServer
	mgr *srkmgr.SrkManager
}

// Package implements a gRPC wrapper around srk.FunctionService.Package().
// chunks represents a stream of bytes representing a tar file containing
// everything that should be included in the function. This tar should extract
// to a single top-level folder. The name of this folder will be used as the
// name of the function.
func (s *srkServer) Package(chunks srkproto.FunctionService_PackageServer) error {

	meta, ok := metadata.FromIncomingContext(chunks.Context())
	if !ok {
		return errors.New("Failed to parse metadata")
	}

	rawName, ok := meta["name"]
	if !ok {
		return errors.New("Metadata option \"name\" is required")
	}
	name := rawName[0]

	includes := meta["includes"]

	// Unpack the uploaded file to a temporary location
	tdir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	// defer os.RemoveAll(tdir)

	funcReader := &pbReader{chunks: chunks}
	_, err = srk.UntarStream(funcReader, tdir)
	if err != nil {
		return errors.Wrap(err, "Could not unpack received tar file")
	}

	// Package the function
	rawDir := s.mgr.GetRawPath(name)

	if err := s.mgr.CreateRaw(tdir, name, includes); err != nil {
		return errors.Wrap(err, "Packaging function failed")
	}
	s.mgr.Logger.Info("Created raw function: " + rawDir)

	pkgPath, err := s.mgr.Provider.Faas.Package(rawDir)
	if err != nil {
		return errors.Wrap(err, "Packaing failed")
	}
	s.mgr.Logger.Info("Package created at: " + pkgPath)

	return nil
}

// Creates a new srk manager (interface to SRK). Be sure to call mgr.Destroy()
// to clean up (failure to do so may require manual cleanup for open-lambda)
func getMgr() *srkmgr.SrkManager {
	mgrArgs := map[string]interface{}{}
	mgrArgs["config-file"] = "./srk.yaml"
	srkLogger := logrus.New()
	srkLogger.SetLevel(logrus.InfoLevel)
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
	srkproto.RegisterFunctionServiceServer(grpcServer, &srkServer{mgr: getMgr()})
	fmt.Println("Server ready")
	grpcServer.Serve(listener)
}
