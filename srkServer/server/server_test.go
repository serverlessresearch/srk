package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/serverlessresearch/srk/srkServer/srkproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

//Number of bytes to return per Recv. It is intentionally not a round number.
const chunkSize = 60

const bufConnSize = 1024 * 1024

type dummyPackageServer struct {
	// Here just to implement FunctionService_PackageServer interface
	grpc.ServerStream

	// Backing buffer used for dummy implementation
	buf   []byte
	index int

	// Indicates user has called SendAndClose
	closed bool
}

func (x *dummyPackageServer) SendAndClose(m *srkproto.PackageRet) error {
	if x.index != len(x.buf) {
		return errors.New("Did not read entire stream")
	}

	x.closed = true
	return nil
}

func (x *dummyPackageServer) Recv() (*srkproto.ByteTransfer, error) {
	if x.index == len(x.buf) {
		return nil, io.EOF
	}

	var toRead = 0
	if len(x.buf)-x.index < chunkSize {
		toRead = len(x.buf) - x.index
	} else {
		toRead = chunkSize
	}
	chunk := x.buf[x.index : x.index+toRead]
	x.index += toRead
	return &srkproto.ByteTransfer{Chunk: chunk}, nil
}

// Testing the tester
func TestDummyPackageServer(t *testing.T) {
	testSize := 1050
	data := make([]byte, testSize)
	_, err := rand.Read(data)
	if err != nil {
		t.Errorf("Failed to create random test array: %v\n", err)
	}

	c := dummyPackageServer{buf: data}

	nread := 0
	for rcvBuf, err := c.Recv(); err != io.EOF; rcvBuf, err = c.Recv() {
		if err != nil {
			t.Fatalf("Failed to read from dummyPackageServer: %v\n", err)
		}

		for i := 0; i < len(rcvBuf.Chunk); i++ {
			if rcvBuf.Chunk[i] != data[nread+i] {
				t.Fatalf("Received data doesn't match original at index %v: expected %v, got %v\n", nread+i, data[nread+i], rcvBuf.Chunk[i])
			}
		}
		nread += len(rcvBuf.Chunk)
	}

	if nread != testSize {
		t.Fatalf("Did not read entire backing buffer: expected %v, got %v\n", testSize, nread)
	}
	err = c.SendAndClose(&srkproto.PackageRet{})
}

func TestPbReader(t *testing.T) {
	testSize := 1050
	data := make([]byte, testSize)
	_, err := rand.Read(data)
	if err != nil {
		t.Errorf("Failed to create random test array: %v\n", err)
	}

	packServer := dummyPackageServer{buf: data}

	r := &pbReader{chunks: &packServer}

	tBuf := make([]byte, 1)
	n, err := r.Read(tBuf)
	if err != nil {
		t.Fatalf("Failed to read first byte of buffer: %v\n", err)
	}

	if n != 1 {
		t.Fatalf("Read too many bytes: expected 1, got %v\n", n)
	}

	if tBuf[0] != data[0] {
		t.Fatalf("First byte returned by reader does not match original: Expected %v, Got %v\n", data[0], tBuf[0])
	}

	allBuf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to ReadAll the buffer: %v\n", err)
	}

	if len(allBuf) != testSize-1 {
		t.Fatalf("ReadAll did not read the right number of bytes: Expected %v, Got %v\n", testSize-1, len(allBuf))
	}

	for i := 0; i < len(allBuf); i++ {
		if allBuf[i] != data[i+1] {
			t.Fatalf("Received data doesn't match original at index %v: expected %v, got %v\n", i+1, data[i+1], allBuf[i])
		}
	}
}

var listener *bufconn.Listener

func bufDialer(string, time.Duration) (net.Conn, error) {
	return listener.Dial()
}

func TestCopyFile(t *testing.T) {
	s := grpc.NewServer()
	srkproto.RegisterTestServiceServer(s, &srkServer{})
	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	defer s.GracefulStop()

	conn, err := grpc.Dial("bufnet", grpc.WithDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Could not connect to bufnet: %v\n", err)
	}

	c := srkproto.NewTestServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	fmt.Println("Sending RPC:")
	r, err := c.CopyFile(ctx, &srkproto.CopyFileArg{Src: "./t1", Dst: "./t2"})
	if err != nil {
		log.Fatalf("RPC failed: %v\n", err)
	}
	fmt.Printf("Response: %v\n", r)
	fmt.Println("Test Success")

}

func newFunctionServiceServer() (*grpc.Server, error) {
	s := grpc.NewServer()
	srkproto.RegisterFunctionServiceServer(s, &srkServer{})
	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	return s, nil
}

func newFunctionServiceClient() (srkproto.FunctionServiceClient, error) {
	conn, err := grpc.Dial("bufnet", grpc.WithDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return srkproto.NewFunctionServiceClient(conn), nil
}

func sendStream(sendStream srkproto.FunctionService_PackageClient, stream io.Reader) error {
	for {
		buf := make([]byte, chunkSize)
		nread, err := stream.Read(buf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		sendStream.Send(&srkproto.ByteTransfer{Chunk: buf[:nread]})
	}
}

func TestPackage(t *testing.T) {
	// Create the server
	s, err := newFunctionServiceServer()
	if err != nil {
		t.Fatalf("Failed to create FunctionServiceServer: %v\n", err)
	}
	defer s.GracefulStop()

	// Create the client
	c, err := newFunctionServiceClient()
	if err != nil {
		t.Fatalf("Failed to get function service client: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := c.Package(ctx)
	if err != nil {
		t.Fatalf("Failed to invoke rpc Package: %v\n", err)
	}

	tFile, err := os.Open("testData/t1")
	if err != nil {
		t.Fatalf("Couldn't open test data 'testData/t1': %v", err)
	}
	defer tFile.Close()

	err = sendStream(stream, tFile)
	if err != nil {
		t.Errorf("Failed to send file: %v\n", err)
	}

	_, err = stream.CloseAndRecv()
	if err != nil && err != io.EOF {
		t.Errorf("Failed to close stream: %v\n", err)
	}
}

func TestMain(m *testing.M) {
	// Set up the buffered connection to be used by all tests
	listener = bufconn.Listen(bufConnSize)

	code := m.Run()
	os.Exit(code)
}
