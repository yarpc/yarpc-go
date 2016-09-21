package grpc

import (
	"bytes"
	"time"

	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/internal"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var grpcOptions transport.Options

type handler struct {
	Handler transport.Handler
	Deps    transport.Deps
}

// Handle the grpc request and convert it into a YARPC request
// dec ('decode') will pass through the request body in raw bytes using the PassThroughCodec
func (h handler) Handle(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	var requestBody []byte
	if err := dec(&requestBody); err != nil {
		return nil, err
	}

	requestBodyBuffer := bytes.NewBuffer(requestBody)

	// TODO replace the hardcoded strings with Headers from the request
	treq := &transport.Request{
		Caller:    "hello",
		Service:   "foo",
		Procedure: "bar",
		Encoding:  "raw",
		Body:      requestBodyBuffer,
	}

	// TODO handle deadlines
	// TODO handle validation

	start := time.Now()

	var r response
	rw := newResponseWriter(&r)

	err := internal.SafelyCallHandler(h.Handler, start, ctx, grpcOptions, treq, rw)

	responseBody := r.body.Bytes()
	return &responseBody, err
}

// The response object contains response information from the YARPC handler
type response struct {
	body    bytes.Buffer
	headers transport.Headers
}

// Wrapper to control writes to the response object
type responseWriter struct {
	r *response
}

func newResponseWriter(r *response) responseWriter {
	return responseWriter{r: r}
}

func (rw responseWriter) Write(s []byte) (int, error) {
	return rw.r.body.Write(s)
}

func (rw responseWriter) AddHeaders(h transport.Headers) {
	rw.r.headers = h
}

func (responseWriter) SetApplicationError() {
	// Nothing to do.
}
