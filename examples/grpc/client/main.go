package main

import (
	"fmt"
	"log"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
)

const (
	address = "localhost:50054"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithCodec(testCodec{}))
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

type testCodec struct {
}

func (testCodec) Marshal(v interface{}) ([]byte, error) {
	fmt.Println("client.go-testCodec::Marshal")
	return []byte(*(v.(*string))), nil
}

func (testCodec) Unmarshal(data []byte, v interface{}) error {
	fmt.Println("client.go-testCodec::Unmarshal")
	*(v.(*string)) = string(data)
	return nil
}

func (testCodec) String() string {
	return "test"
}
