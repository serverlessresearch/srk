package main

import (
	"context"
	"os"
	"path"
	"io/ioutil"
	"flag"
	"fmt"
	"log"
	"net"
	"math"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/serverlessresearch/srk/pkg/objstore"
)

type localObjStore struct {
	storageDir string
}

func errorHandler (err error) (error) {
	if os.IsExist(err) {
		return status.Error(codes.AlreadyExists, "entity already exists")
	} else if os.IsNotExist(err) {
		return status.Error(codes.NotFound, "requested entity was not found")
	} else if os.IsPermission(err) {
		return status.Error(codes.PermissionDenied, "permission denied")
	} else {
		return status.Error(codes.Unknown, "unknown error")
	}
}

func (o *localObjStore) CreateBucket(ctx context.Context, r *pb.CreateBucketRequest) (*empty.Empty, error) {
	err := os.Mkdir(path.Join(o.storageDir, r.GetBucketName()), 0755)
	if err != nil {
		// TODO: is this the right way to return error messages over gRPC?
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

func (o *localObjStore) ListBucket(ctx context.Context, r *pb.ListBucketRequest) (*pb.ListBucketResponse, error) {
	files, err := ioutil.ReadDir(path.Join(o.storageDir, r.GetBucketName()))
	if err != nil {
		return nil, errorHandler(err)
	}
	names := make([]string, len(files))
	for i := 0; i < len(files); i++ {
		names[i] = files[i].Name()
	}
	return &pb.ListBucketResponse{ObjectName:names}, nil
}

func (o *localObjStore) DeleteBucket(ctx context.Context, r *pb.DeleteBucketRequest) (*empty.Empty, error) {
	err := os.RemoveAll(path.Join(o.storageDir, r.GetBucketName()))
	if err != nil {
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

func (o *localObjStore) Get(ctx context.Context, r *pb.GetRequest) (*pb.GetResponse, error) {
	data, err := ioutil.ReadFile(path.Join(o.storageDir, r.GetBucketName(), r.GetObjectName()))
	if err != nil {
		return nil, errorHandler(err)
	}
	return &pb.GetResponse{Data: data}, nil
}

func (o *localObjStore) Put(ctx context.Context, r *pb.PutRequest) (*empty.Empty, error) {
	err := ioutil.WriteFile(path.Join(o.storageDir, r.GetBucketName(), r.GetObjectName()), r.GetData(), 0644) 
	if err != nil {
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

func (o *localObjStore) Delete(ctx context.Context, r *pb.DeleteRequest) (*empty.Empty, error) {
	err := os.Remove(path.Join(o.storageDir, r.GetBucketName(), r.GetObjectName()))
	if err != nil {
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

func newServer(storageDir string) *localObjStore {
	s := &localObjStore{storageDir: storageDir}
	return s
}

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	port       = flag.Int("port", 10000, "The server port")
)

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

	opts = append(opts, grpc.MaxRecvMsgSize(math.MaxInt32))
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterObjectStoreServer(grpcServer, newServer("/tmp/objfiles"))
	grpcServer.Serve(lis)
}