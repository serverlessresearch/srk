package main

import (
	"context"
	"flag"
	"log"
	"io/ioutil"
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
	objectName		   = flag.String("object_name", "test.obj", "object storage object name")
)

func createBucket(client pb.ObjectStoreClient, bucketName string) {
	_, err := client.CreateBucket(context.Background(), &pb.CreateBucketRequest{BucketName: bucketName})
	if err != nil {
		log.Fatalf("%v.CreateBucket(_) = _, %v: ", client, err)
	}
}

func listBucket(client pb.ObjectStoreClient, bucketName string) {
	res, err := client.ListBucket(context.Background(), &pb.ListBucketRequest{BucketName: bucketName})
	if err != nil {
		log.Fatalf("%v.ListBucket(_) = _, %v: ", client, err)
	} else {
		for _, objectName := range res.GetObjectName() {
			println(objectName)
		}
	}
}

func getObject(client pb.ObjectStoreClient, bucketName string, objectName string) ([]byte) {
	res, err := client.Get(context.Background(), &pb.GetRequest{ BucketName: bucketName, ObjectName: objectName})
	if err != nil {
		log.Fatalf("%v.Get(_) = _, %v: ", client, err)
	}
	return res.GetData()
}

func putObject(client pb.ObjectStoreClient, bucketName string, objectName string, data []byte) {
	_, err := client.Put(context.Background(), &pb.PutRequest{ BucketName: bucketName, ObjectName: objectName, Data: data })
	if err != nil {
		log.Fatalf("%v.Put(_) = _, %v: ", client, err)
	}
}

func uploadFile(client pb.ObjectStoreClient, bucketName string, objectName string, filepath string) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Fail to read file %v: %v: ", filepath, err)
	}
	putObject(client, bucketName, objectName, data)
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

	testString := "Test测试" // word "test" in English and Chinese
	createBucket(client, *bucketName)
	listBucket(client, *bucketName)
	putObject(client, *bucketName, *objectName, []byte(testString))
	println("put string: ", testString)
	data := getObject(client, *bucketName, *objectName)
	println("get string: ", string(data))
}
