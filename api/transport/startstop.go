package transport

// StartStoppable objects are used to define a common Start/Stop functionality
// across different dispatcher objects
type StartStoppable interface {
	// Starts the RPC allowing it to accept and process new incoming
	// requests.
	//
	// Blocks until the RPC is ready to start accepting new requests.
	Start() error

	// Stops the RPC. No new requests will be accepted.
	//
	// Blocks until the RPC has stopped.
	Stop() error
}
