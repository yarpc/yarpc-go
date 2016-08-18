package main

import (
	"fmt"
	"log"

	gr "github.com/yarpc/yarpc-go/transport/grpc"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50054"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithCodec(gr.RawCodec{}))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// typically called by generated code
	req := "hello from client.go!"
	var res string
	err = grpc.Invoke(context.Background(), "/foo/bar", &req, &res, conn) // @generated
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	fmt.Println("GOT RESP:", res)
}
