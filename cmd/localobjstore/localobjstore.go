// LocalObjStore provides a simple filesystem-based implementation of the SRK Object Storage API defined
// in the protocol buffer specification (https://github.com/serverlessresearch/srk/blob/master/pkg/objstore/objstore.proto).
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

// The information necessary for local object store server
type LocalObjStore struct {
	// The storage directory
	storageDir string
}


// Map local file system errors to standard gRPC status codes
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

// Create a bucket by creating a directory whose name is the same as the bucket name on local file system,
// Returns a gRPC error code if failed to create the directory or the directory already exits 
func (o *LocalObjStore) CreateBucket(ctx context.Context, r *pb.CreateBucketRequest) (*empty.Empty, error) {
	err := os.Mkdir(path.Join(o.storageDir, r.GetBucketName()), 0755)
	if err != nil {
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

// List a bucket by listing all the filenames in the directory,
// Return a respons that contains all the names if no error,
func (o *LocalObjStore) ListBucket(ctx context.Context, r *pb.ListBucketRequest) (*pb.ListBucketResponse, error) {
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

// Delete a bucket by erasing the whole directory including all files under it
func (o *LocalObjStore) DeleteBucket(ctx context.Context, r *pb.DeleteBucketRequest) (*empty.Empty, error) {
	// S3 doesn't empty the bucket/directory,
	// Instead it first check whether the bucket is empty or not, throw an error if not
	err := os.RemoveAll(path.Join(o.storageDir, r.GetBucketName()))
	if err != nil {
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

// Get an object by reading the local file and return an response that wraps the content if no error
func (o *LocalObjStore) Get(ctx context.Context, r *pb.GetRequest) (*pb.GetResponse, error) {
	data, err := ioutil.ReadFile(path.Join(o.storageDir, r.GetBucketName(), r.GetObjectName()))
	if err != nil {
		return nil, errorHandler(err)
	}
	return &pb.GetResponse{Data: data}, nil
}

// Put an object by writing bytes of data into a file whose name is the same as the object name.
// It overwrites the object if already exits
func (o *LocalObjStore) Put(ctx context.Context, r *pb.PutRequest) (*empty.Empty, error) {
	err := ioutil.WriteFile(path.Join(o.storageDir, r.GetBucketName(), r.GetObjectName()), r.GetData(), 0644)
	if err != nil {
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

// Delete an object by removing the local file
func (o *LocalObjStore) DeleteObject(ctx context.Context, r *pb.DeleteRequest) (*empty.Empty, error) {
	err := os.Remove(path.Join(o.storageDir, r.GetBucketName(), r.GetObjectName()))
	if err != nil {
		return nil, errorHandler(err)
	}
	return &empty.Empty{}, nil
}

// Initialize a local object store server struct and return its reference
func newServer(storageDir string) *LocalObjStore {
	s := &LocalObjStore{storageDir: storageDir}
	return s
}

// Flag variables for TLS and server port
var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	port       = flag.Int("port", 10000, "The server port")
)

func main() {
	// Parse flags
	flag.Parse()
	// Create a TCP listerner
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

	// Set the max message size in bytes the server can receive to 2147483647 
	opts = append(opts, grpc.MaxRecvMsgSize(math.MaxInt32))
	// Create a new gRPC local object store server
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterObjectStoreServer(grpcServer, newServer("/tmp/objfiles"))
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
