package main

import (
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/serverlessresearch/srk/srkServer/srkproto"
	"google.golang.org/grpc"
)

//Number of bytes to return per Recv. It is intentionally not a round number.
const chunkSize = 60

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
