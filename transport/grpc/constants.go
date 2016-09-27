package grpc

const (
	// ApplicationHeaderPrefix is the prefix added to application headers over
	// the wire.
	ApplicationHeaderPrefix = "rpc-header-"

	// BaggageHeaderPrefix is the prefix added to context headers over the wire.
	BaggageHeaderPrefix = "context-"

	// CallerHeader is the HTTP header used to indiate the service doing the calling
	CallerHeader = "rpc-caller"

	// EncodingHeader is the HTTP header used to specify the name of the
	// encoding (raw, json, thrift, etc).
	EncodingHeader = "rpc-encoding"
)
