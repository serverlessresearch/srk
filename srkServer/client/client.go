package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/serverlessresearch/srk/srkServer/srkproto"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Initiating connection:")
	conn, err := grpc.Dial("localhost:8000", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to dial server: %v\n", err)
	}
	defer conn.Close()
	fmt.Println("connection successful")

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
