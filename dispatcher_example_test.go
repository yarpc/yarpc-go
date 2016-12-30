package yarpc_test

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
)

func ExampleDispatcher_minimal() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{Name: "myFancyService"})
	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()
}

// global dispatcher used in the registration examples
var dispatcher = yarpc.NewDispatcher(yarpc.Config{Name: "service"})

func ExampleDispatcher_Register_raw() {
	handler := func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}

	dispatcher.Register(raw.Procedure("echo", handler))
}

// Excuse the weird naming of this function. This lets is show as "JSON"
// rather than "Json"

func ExampleDispatcher_Register_jSON() {
	handler := func(ctx context.Context, key string) (string, error) {
		fmt.Println("key", key)
		return "value", nil
	}

	dispatcher.Register(json.Procedure("get", handler))
}
