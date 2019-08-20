package main

import (
	"context"
	"flag"
	"log"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	// "google.golang.org/grpc/status"
	"google.golang.org/grpc/testdata"
	pb "github.com/serverlessresearch/srk/pkg/objstore"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
	bucketName		   = flag.String("bucket_name", "/", "Object storage bucket name")
)

func createBucket(client pb.ObjectStoreClient, bucketName string) {
	_, err := client.CreateBucket(context.Background(), &pb.CreateBucketRequest{BucketName: bucketName})
	if err != nil {
		log.Fatalf("%v.CreateBUcket(_) = _, %v: ", client, err)
	}
}

func listBucket(client pb.ObjectStoreClient, bucketName string) {
	listBucketResponse, err := client.ListBucket(context.Background(), &pb.ListBucketRequest{BucketName: bucketName})
	if err != nil {
		log.Fatalf("%v.ListBUcket(_) = _, %v: ", client, err)
	} else {
		for _, objectName := range listBucketResponse.GetObjectName() {
			println(objectName)
		}
	}
}

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = testdata.Path("ca.pem")
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewObjectStoreClient(conn)

	createBucket(client, *bucketName)
	listBucket(client, *bucketName)
}
