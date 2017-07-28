[← Errors][back] - [:book:][index] - [Configuring Transports →][next]

# Binary Encodings

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur eget commodo magna. Nam varius, purus tincidunt pulvinar auctor, sapien lorem consequat tortor, in cursus enim sapien vel erat. Donec non sollicitudin neque. Integer rutrum egestas purus ac porttitor. Morbi maximus metus dapibus iaculis molestie. Nam euismod sed est quis auctor. Aenean at odio vitae orci sodales lobortis nec a nibh. Maecenas condimentum ipsum risus, eu pellentesque augue facilisis a. In hac habitasse platea dictumst. Quisque fermentum accumsan felis, eu venenatis nisl pulvinar vel. Nam eu accumsan est. Sed viverra vel ligula sit amet placerat. Donec molestie ut mauris nec egestas.

## Thrift

Walk end-user through building the thrift-hello example.

## Protobuf

Install protoc and YARPC's proto plugin:

```
$ go get github.com/gogo/protobuf/protoc-gen-gogoslick
$ go get go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go
```

Author a protobuf IDL:

```proto
syntax = "proto3";

package hello;

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
}

service HelloWorld {
  rpc Hello(HelloRequest) returns (HelloResponse);
}
```

Now generate the stubs:

```
$ protoc --gogoslick_out=. --yarpc-go_out=. hello.proto
```

This will generate YARPC interfaces in `hello.pb.yarpc.go`:

```go
type HelloWorldYARPCClient interface {
	Hello(context.Context, *HelloRequest, ...yarpc.CallOption) (*HelloResponse, error)
}

type HelloWorldYARPCServer interface {
	Hello(context.Context, *HelloRequest) (*HelloResponse, error)
}
```

Implement the server interface in `server/main.go`:

```go
type handler struct{}

func (handler) Hello(context.Context, *hello.HelloRequest) (*hello.HelloResponse, error) {
	message := fmt.Sprintf("Hello %s!", hello.HelloRequest.Name)
	return &hello.HelloResponse{Message: message}, nil
}
```

Install the handler in a Dispatcher:

```go
// build a configurator with the HTTP transport registered
configurator := config.New()
configurator.MustRegisterTransport(http.TransportSpec())

// create a dispatcher for the "server" service
// TODO switch to yaml
dispatcher, err := configurator.NewDispatcherFromYAML("server", strings.NewReader(`{
    "inbounds": {
        "http": {
            "address": ":8080",
        },
    },
}`))
if err != nil {
    log.Panicf("Dispatcher could not be created: %v", err)
}

// register handler
procedures := hello.BuildHelloWorldYARPCProcedures(handler{})
dispatcher.Register(procedures)

// start service
dispatcher.Start()
defer dispatcher.Stop()

// block until SIGINT/SIGTERM
signals := make(chan os.Signal, 1)
signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
<-signals
```

Start the server:

```
$ go run ./server/main.go
```

Curl the server:

```
$ curl -s localhost:8080 -X POST \
-H RPC-Service:server \
-H RPC-Procedure:hello.HelloWorld::Hello \
-H RPC-Caller:curl \
-H RPC-Encoding:json \
-H Context-TTL-MS:10000 \
-d '{"name": "Grayson"}' | jq .
{
  "message": "Hello Grayson!"
}
```

Now, instead of using curl, create another service which calls "server":

```go
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	hello "go.uber.org/yarpc/internal/examples/protobuf-hello"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/x/config"
)

const yaml = `
outbounds:
  server:
    http:
      url: "http://localhost:8080/"
`

func main() {
	// build a configurator with the HTTP transport registered
	configurator := config.New()
	configurator.MustRegisterTransport(http.TransportSpec())

	// create a dispatcher for the "client" service
	dispatcher, err := configurator.NewDispatcherFromYAML("client", strings.NewReader(yaml))
	if err != nil {
		log.Panicf("Dispatcher could not be created: %v", err)
	}

	// create a client for the "server" service
	client := hello.NewHelloWorldYARPCClient(dispatcher.ClientConfig("server"))

	// start service
	dispatcher.Start()
	defer dispatcher.Stop()

	// prepare a context to call
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// make an rpc to the "server" service
	resp, err := client.Hello(ctx, &hello.HelloRequest{Name: "client"})
	if err != nil {
		log.Panicf("Could not call server: %v", err)
	}

	fmt.Printf("Called server successfully: %v\n", resp.Message)
}
```

Now run `client/main.go`:

```
$ go run ./client/main.go
Called server successfully: Hello client!
```

Cheers.

[← Errors][back] - [:book:][index] - [Configuring Transports →][next]

[index]: /README.md#usage
[back]: 07-errors.md
[next]: 09-configuring-transports.md
