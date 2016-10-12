package grpc

const (
	// ApplicationHeaderPrefix is the prefix added to application headers over
	// the wire.
	ApplicationHeaderPrefix = "rpc-header-"

	// CallerHeader is the HTTP header used to indiate the service doing the calling
	CallerHeader = "rpc-caller"

	// EncodingHeader is the HTTP header used to specify the name of the
	// encoding (raw, json, thrift, etc).
	EncodingHeader = "rpc-encoding"
)
