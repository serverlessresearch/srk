package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"
	"fmt"
	"strconv"
	"io/ioutil"
	"sync/atomic"
	"net/http"
	"encoding/json"
	"google.golang.org/grpc"
	"code.cloudfoundry.org/bytefmt"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/testdata"
	pb "github.com/serverlessresearch/srk/pkg/objstore"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
	
	bucketName		   = flag.String("bucket_name", "/", "Object storage bucket name")
	durationSecs	   = flag.Int("duration", 10, "Duration of each test in seconds")
	size               = flag.String("size", "1M", "Size of objects in bytes with postfix K, M, and G")
	threads  		   = flag.Int("threads", 1, "Number of threads to run")
	loops			   = flag.Int("loops", 1, "Number of times to repeat test")
)

func createBucket(client pb.ObjectStoreClient, bucketName string) {
	_, err := client.CreateBucket(context.Background(), &pb.CreateBucketRequest{BucketName: bucketName})
	if err != nil {
		errStatus, _ := status.FromError(err)
		log.Fatalf("%v.CreateBucket(_) = _, code = %v msg = %v", client, errStatus.Code(), errStatus.Message())
	}
}

func listBucket(client pb.ObjectStoreClient, bucketName string) {
	res, err := client.ListBucket(context.Background(), &pb.ListBucketRequest{BucketName: bucketName})
	if err != nil {
		errStatus, _ := status.FromError(err)
		log.Fatalf("%v.ListBucket(_) = _, code = %v msg = %v", client, errStatus.Code(), errStatus.Message())
	}
	for _, objectName := range res.GetObjectName() {
		println(objectName)
	}
}

func deleteBucket(client pb.ObjectStoreClient, bucketName string) {
	_, err := client.DeleteBucket(context.Background(), &pb.DeleteBucketRequest{BucketName: bucketName})
	if err != nil {
		errStatus, _ := status.FromError(err)
		log.Fatalf("%v.DeleteBucket(_) = _, code = %v msg = %v", client, errStatus.Code(), errStatus.Message())
	}
}

func getObject(client pb.ObjectStoreClient, bucketName string, objectName string) ([]byte) {
	res, err := client.Get(context.Background(), &pb.GetRequest{ BucketName: bucketName, ObjectName: objectName})
	if err != nil {
		errStatus, _ := status.FromError(err)
		log.Fatalf("%v.Get(_) = _, code = %v msg = %v", client, errStatus.Code(), errStatus.Message())
	}
	return res.GetData()
}

func putObject(client pb.ObjectStoreClient, bucketName string, objectName string, data []byte) {
	_, err := client.Put(context.Background(), &pb.PutRequest{ BucketName: bucketName, ObjectName: objectName, Data: data })
	if err != nil {
		errStatus, _ := status.FromError(err)
		log.Fatalf("%v.Put(_) = _, code = %v msg = %v", client, errStatus.Code(), errStatus.Message())
	}
}

func uploadFile(client pb.ObjectStoreClient, bucketName string, objectName string, filepath string) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		errStatus, _ := status.FromError(err)
		log.Fatalf("Fail to read the file: code = %v msg = %v", errStatus.Code(), errStatus.Message())
	}
	putObject(client, bucketName, objectName, data)
}

func runUpload(threadNum int, client pb.ObjectStoreClient, bucketName string) {
	for time.Now().Before(endtime) {
		objnum := atomic.AddInt64(&uploadCount, 1)
		putObject(client, bucketName, strconv.FormatInt(objnum, 10), objectData)
	}
	// Remember last done time
	uploadFinish = time.Now()
	// One less thread
	atomic.AddInt64(&runningThreads, -1)
}

func runDownload(threadNum int, client pb.ObjectStoreClient, bucketName string) {
	for time.Now().Before(endtime) {
		atomic.AddInt64(&downloadCount, 1)
		objnum := rand.Int63n(uploadCount) + 1
		getObject(client, bucketName, strconv.FormatInt(objnum, 10))
	}
	// Remember last done time
	downloadFinish = time.Now()
	// One less thread
	atomic.AddInt64(&runningThreads, -1)
}

func runDelete(threadNum int, client pb.ObjectStoreClient, bucketName string) {
	for {
		objnum := atomic.AddInt64(&deleteCount, 1)
		if objnum > uploadCount {
			break
		}
		deleteObject(client, bucketName, strconv.FormatInt(objnum, 10))
	}
	// Remember last done time
	deleteFinish = time.Now()
	// One less thread
	atomic.AddInt64(&runningThreads, -1)
}

func deleteObject(client pb.ObjectStoreClient, bucketName string, objectName string) { 
	_, err := client.Delete(context.Background(), &pb.DeleteRequest{ BucketName: bucketName, ObjectName: objectName})
	if err != nil {
		errStatus, _ := status.FromError(err)
		log.Fatalf("%v.Delete(_) = _, code = %v msg = %v", client, errStatus.Code(), errStatus.Message())
	}
}

// Global variables
var objectData []byte
var runningThreads, uploadCount, downloadCount, deleteCount int64
var endtime, uploadFinish, downloadFinish, deleteFinish time.Time

type logMessage struct {
	LogTime    time.Time `json:"time"`
	Method     string    `json:"method"`
	Loop       int       `json:"loop"`
	Time       float64   `json:"timeTaken"`
	Objects    int64     `json:"totalObjects"`
	Speed      string    `json:"avgSpeed"`
	RawSpeed   uint64    `json:"rawSpeed"`
	Operations float64   `json:"totalOperations"`
}

func (l logMessage) String() string {
	if l.Speed != "" {
		return fmt.Sprintf("%s Loop %d: %s time %.1f secs, objects = %d, speed = %sB/sec, %.1f operations/sec.",
			l.LogTime.Format(http.TimeFormat), l.Loop, l.Method, l.Time, l.Objects, l.Speed, l.Operations)
	}
	return fmt.Sprintf("%s Loop %d: %s time %.1f secs, %.1f operations/sec.",
		l.LogTime.Format(http.TimeFormat), l.Loop, l.Method, l.Time, l.Operations)
}

func (l logMessage) JSON() string {
	data, err := json.Marshal(&l)
	if err != nil {
		panic(err)
	}
	return string(data)
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

	maxMsgSize := 1024*1024*1024
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)))
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewObjectStoreClient(conn)

	var objectSize uint64
	if objectSize, err = bytefmt.ToBytes(*size); err != nil {
		log.Fatalf("Invalid -z argument for object size: %v", err)
	}

	fmt.Println(fmt.Sprintf("Parameters: bucket=%s, duration=%d seconds, threads=%d, loops=%d, size=%s", *bucketName, *durationSecs, *threads, *loops, *size))
	// Initialize data for the bucket
	objectData = make([]byte, objectSize)
	rand.Read(objectData)

	// Create the bucket
	createBucket(client, *bucketName) 

	// Loop running the tests
	for loop := 1; loop <= *loops; loop++ {
		uploadCount = 0
		downloadCount = 0
		// Run the upload case
		runningThreads = int64(*threads)
		starttime := time.Now()
		endtime = starttime.Add(time.Second * time.Duration(*durationSecs))
		for n := 1; n <= *threads; n++ {
			go runUpload(n, client, *bucketName)
		}

		// Wait for it to finish
		for atomic.LoadInt64(&runningThreads) > 0 {
			time.Sleep(time.Millisecond)
		}
		uploadTime := uploadFinish.Sub(starttime).Seconds()

		bps := float64(uint64(uploadCount)*objectSize) / uploadTime
		
		l := logMessage{
			LogTime:    time.Now(),
			Loop:       loop,
			Method:     http.MethodPut,
			Time:       uploadTime,
			Objects:    uploadCount,
			Speed:      bytefmt.ByteSize(uint64(bps)),
			RawSpeed:   uint64(bps), 
			Operations: (float64(uploadCount) / uploadTime),
		}
		fmt.Println(l.String())

		// Run the download case
		runningThreads = int64(*threads)
		starttime = time.Now()
		endtime = starttime.Add(time.Second * time.Duration(*durationSecs))
		for n := 1; n <= *threads; n++ {
			go runDownload(n, client, *bucketName)
		}

		// Wait for it to finish
		for atomic.LoadInt64(&runningThreads) > 0 {
			time.Sleep(time.Millisecond)
		}
		downloadTime := downloadFinish.Sub(starttime).Seconds()

		bps = float64(uint64(downloadCount)*objectSize) / downloadTime
		l = logMessage{
			LogTime:    time.Now(),
			Loop:       loop,
			Method:     http.MethodGet,
			Time:       downloadTime,
			Objects:    downloadCount,
			Speed:      bytefmt.ByteSize(uint64(bps)),
			RawSpeed:   uint64(bps),
			Operations: (float64(downloadCount) / downloadTime),
		}
		fmt.Println(l.String())

		// Run the delete case
		runningThreads = int64(*threads)
		starttime = time.Now()
		endtime = starttime.Add(time.Second * time.Duration(*durationSecs))
		for n := 1; n <= *threads; n++ {
			go runDelete(n, client, *bucketName)
		}

		// Wait for it to finish
		for atomic.LoadInt64(&runningThreads) > 0 {
			time.Sleep(time.Millisecond)
		}
		deleteTime := deleteFinish.Sub(starttime).Seconds()

		l = logMessage{
			LogTime:    time.Now(),
			Loop:       loop,
			Method:     http.MethodDelete,
			Time:       deleteTime,
			Operations: (float64(uploadCount) / deleteTime),
		}
		fmt.Println(l.String())
	}

	deleteBucket(client, *bucketName)
	// All done
	fmt.Println("Benchmark completed.")
}
