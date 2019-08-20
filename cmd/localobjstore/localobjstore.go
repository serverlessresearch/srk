package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	// "google.golang.org/grpc/status"
	"google.golang.org/grpc/testdata"
	pb "github.com/serverlessresearch/srk/pkg/objstore"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
)

type localObjStore struct {
	storageDir string
}

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	port       = flag.Int("port", 10000, "The server port")
)

func (o *localObjStore) CreateBucket(ctx context.Context, r *pb.CreateBucketRequest) (*pb.Empty, error) {
	err := os.Mkdir(path.Join(o.storageDir, r.BucketName), 0755)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (o *localObjStore) ListBucket(ctx context.Context, r *pb.ListBucketRequest) (*pb.ListBucketResponse, error) {
	files, err := ioutil.ReadDir(path.Join(o.storageDir, r.BucketName))
	if err != nil {
		return nil, err
	}
	names := make([]string, len(files))
	for i := 0; i < len(files); i++ {
		names[i] = files[i].Name()
	}
	return &pb.ListBucketResponse{ObjectName:names}, nil
}


func newServer(storageDir string) *localObjStore {
	s := &localObjStore{storageDir: storageDir}
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("Listening on port %d\n", *port)
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = testdata.Path("server1.pem")
		}
		if *keyFile == "" {
			*keyFile = testdata.Path("server1.key")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterObjectStoreServer(grpcServer, newServer("/tmp/objfiles"))
	grpcServer.Serve(lis)
}