# YARPC Reflection

This package implements gRPC reflection for YARPC servers, allowing clients to discover and inspect service definitions at runtime. It provides a way for clients to query information about the services, methods, and message types available on a YARPC server.

## Overview

The reflection package implements the [gRPC reflection protocol](https://grpc.io/docs/guides/reflection/) (v1alpha) for YARPC servers. It enables:

- Service discovery: Clients can list all available services
- File descriptor retrieval: Clients can get the Protocol Buffer file descriptors for services
- Symbol lookup: Clients can find file descriptors containing specific symbols
- Extension information: Clients can query extension numbers and their containing types

## Usage

To enable reflection on your YARPC server, you need to:

1. Create reflection procedures using the `NewServer` function:

```go
import (
    "go.uber.org/yarpc/x/reflection"
    "go.uber.org/yarpc/encoding/protobuf/reflection"
)

// Create reflection procedures
procedures, err := reflection.NewServer([]reflection.ServerMeta{
    {
        ServiceName: "your.service.Name",
        FileDescriptors: yourFileDescriptors,
    },
})
if err != nil {
    // Handle error
}
```

2. Add the reflection procedures to your dispatcher:

```go
// Add the procedures to your dispatcher
dispatcher.Register(procedures)
```

After completing these steps, the reflection service will be available at `grpc.reflection.v1alpha.ServerReflection`.
