package transport

// Lifecycle objects are used to define a common Start/Stop functionality
// across different transport objects
type Lifecycle interface {
	// Start the lifecycle object, returns an error if it cannot be started
	// Start MUST be idempotent
	Start() error

	// Stop the lifecycle object, returns an error if it cannot be stopped
	// Stop MUST be idempotent
	Stop() error
}
