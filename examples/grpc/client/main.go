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
	// In the "true" requests from gRPC to YARPC (as opposed to YARPC to YARPC) we will likely get a
	// protobuf encoding, for now, using the PassThroughCodec is necessary to test (though it won't be used
	// in production)
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithCodec(gr.PassThroughCodec{}))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// typically called by generated code
	strReq := "hello from client.go!"
	req := []byte(strReq)

	var res []byte

	err = grpc.Invoke(context.Background(), "/foo/bar", &req, &res, conn) // @generated
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	fmt.Print("GOT RESP:", string(res))
}
