package grpc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	gtransport "google.golang.org/grpc/transport"
)

var grpcOptions transport.Options

type handler struct {
	Registry transport.Registry
	Deps     transport.Deps
}

// Handle the grpc request and convert it into a YARPC request
// dec ('decode') will pass through the request body in raw bytes using the passThroughCodec
func (h handler) Handle(
	srv interface{},
	ctx context.Context,
	dec func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
) (interface{}, error) {
	treq, err := getTRequest(ctx, dec)
	if err != nil {
		return nil, err
	}

	// TODO handle deadlines
	// TODO handle validation

	start := time.Now()
	return callHandler(h, start, ctx, treq)
}

func getTRequest(ctx context.Context, msgBodyDecoder func(interface{}) error) (*transport.Request, error) {
	service, procedure, err := getServiceAndProcedure(ctx)
	if err != nil {
		return nil, err
	}

	requestBody, err := getMsgBody(msgBodyDecoder)
	if err != nil {
		return nil, err
	}

	treq := &transport.Request{
		Service:   service,
		Procedure: procedure,
		Caller:    "yarpc",
		Encoding:  transport.Encoding("raw"),
		Body:      requestBody,
	}
	return treq, nil
}

func getServiceAndProcedure(ctx context.Context) (string, string, error) {
	stream, ok := gtransport.StreamFromContext(ctx)
	if !ok {
		return "", "", errors.New("could not extract stream information from context")
	}
	return getServiceAndProcedureFromMethod(stream.Method())
}

func getServiceAndProcedureFromMethod(streamMethod string) (string, string, error) {
	if streamMethod == "" {
		return "", "", errors.New("no service procedure provided")
	}

	if streamMethod[0] == '/' {
		streamMethod = streamMethod[1:]
	}
	splitPos := strings.LastIndex(streamMethod, "/")

	escapedService := streamMethod[:splitPos]
	service, err := url.QueryUnescape(escapedService)
	if err != nil {
		return "", "", fmt.Errorf("could not parse service for request: %s, error: %v", escapedService, err)
	}

	escapedProcedure := streamMethod[splitPos+1:]
	procedure, err := url.QueryUnescape(escapedProcedure)
	if err != nil {
		return "", "", fmt.Errorf("could not parse procedure for request: %s, error: %v", escapedProcedure, err)
	}

	return service, procedure, nil
}

func getMsgBody(msgBodyDecoder func(interface{}) error) (io.Reader, error) {
	var requestBody []byte
	if err := msgBodyDecoder(&requestBody); err != nil {
		return nil, err
	}
	return bytes.NewBuffer(requestBody), nil
}

func callHandler(
	h handler,
	start time.Time,
	ctx context.Context,
	treq *transport.Request,
) (interface{}, error) {
	var r response
	rw := newResponseWriter(&r)

	handler, err := h.Registry.GetHandler(treq.Service, treq.Procedure)
	if err != nil {
		return nil, err
	}

	err = internal.SafelyCallHandler(handler, start, ctx, grpcOptions, treq, rw)

	responseBody := r.body.Bytes()
	return &responseBody, err
}

// The response object contains response information from the YARPC handler
type response struct {
	body bytes.Buffer
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
	// TODO support Headers
}

func (responseWriter) SetApplicationError() {
	// Nothing to do.
}
