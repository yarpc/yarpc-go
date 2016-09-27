package grpc

import (
	"bytes"
	"time"

	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/internal"

	"strings"

	"fmt"

	"io"

	"errors"

	"net/url"

	"github.com/yarpc/yarpc-go/internal/baggage"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	gtransport "google.golang.org/grpc/transport"
)

var grpcOptions transport.Options

type handler struct {
	Handler transport.Handler
	Deps    transport.Deps
}

// Handle the grpc request and convert it into a YARPC request
// dec ('decode') will pass through the request body in raw bytes using the PassThroughCodec
func (h handler) Handle(
	srv interface{},
	ctx context.Context,
	dec func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
) (interface{}, error) {
	treq, extractErr := getTRequest(ctx, dec)
	if extractErr != nil {
		return nil, extractErr
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

	ctxMetadata, err := getContextMetadata(ctx)
	if err != nil {
		return nil, err
	}

	caller, err := getCaller(ctxMetadata)
	if err != nil {
		return nil, err
	}

	encoding, err := getEncoding(ctxMetadata)
	if err != nil {
		return nil, err
	}

	baggageHeaders := baggageHeaders.FromGRPCMetadata(ctxMetadata, transport.Headers{})
	if baggageHeaders.Len() > 0 {
		// Baggage headers get propagated between request hops by piggybacking on the context
		ctx = baggage.NewContextWithHeaders(ctx, baggageHeaders.Items())
	}

	appHeaders := applicationHeaders.FromGRPCMetadata(ctxMetadata, transport.Headers{})

	requestBody, err := getMsgBody(msgBodyDecoder)
	if err != nil {
		return nil, err
	}

	treq := &transport.Request{
		Service:   service,
		Procedure: procedure,
		Caller:    caller,
		Encoding:  transport.Encoding(encoding),
		Headers:   appHeaders,
		Body:      requestBody,
	}

	return treq, nil
}

func getMsgBody(msgBodyDecoder func(interface{}) error) (io.Reader, error) {
	var requestBody []byte
	if err := msgBodyDecoder(&requestBody); err != nil {
		return nil, err
	}

	requestBodyBuffer := bytes.NewBuffer(requestBody)

	return requestBodyBuffer, nil
}

func getServiceAndProcedure(ctx context.Context) (service, procedure string, err error) {
	stream, ok := gtransport.StreamFromContext(ctx)
	if !ok {
		return "", "", errors.New("Could not extract stream information from context")
	}

	streamMethod := stream.Method()
	if streamMethod != "" && streamMethod[0] == '/' {
		streamMethod = streamMethod[1:]
	}
	splitPos := strings.LastIndex(streamMethod, "/")

	service, err = url.QueryUnescape(streamMethod[:splitPos])
	if err != nil {
		return "", "", fmt.Errorf("Could not parse service for request: %s, error: %v", streamMethod[:splitPos], err)
	}

	procedure, err = url.QueryUnescape(streamMethod[splitPos+1:])
	if err != nil {
		return "", "", fmt.Errorf("Could not parse procedure for request: %s, error: %v", streamMethod[splitPos+1:], err)
	}
	return
}

func getContextMetadata(ctx context.Context) (metadata.MD, error) {
	contextMetadata, ok := metadata.FromContext(ctx)
	if !ok || contextMetadata == nil {
		return nil, errors.New("Could not extract metadata information from context")
	}
	return contextMetadata, nil
}

func getCaller(ctxMetadata metadata.MD) (string, error) {
	// TODO Make a gRPC outbounds require a header for caller and enforce it at inbounds
	return extractMetadataHeader(ctxMetadata, CallerHeader)
}

func getEncoding(ctxMetadata metadata.MD) (string, error) {
	// TODO Add a pull request to gRPC to add encoding via content-type to contexts so we don't have to set it manually ourselves
	return extractMetadataHeader(ctxMetadata, EncodingHeader)
}

func extractMetadataHeader(ctxMetadata metadata.MD, header string) (string, error) {
	headerList, ok := ctxMetadata[header]
	if !ok {
		return "", fmt.Errorf("Couldn't extract header:(%s) from Context Metadata (%v)", header, ctxMetadata)
	}
	if len(headerList) != 1 {
		return "", fmt.Errorf("Invalid number of headers for %s, expected 1, got %d", header, len(headerList))
	}
	return headerList[0], nil
}

func callHandler(
	h handler,
	start time.Time,
	ctx context.Context,
	treq *transport.Request,
) (interface{}, error) {
	var r response
	rw := newResponseWriter(&r)

	err := internal.SafelyCallHandler(h.Handler, start, ctx, grpcOptions, treq, rw)

	// Sends the headers back on the request
	grpc.SendHeader(ctx, r.headers)
	responseBody := r.body.Bytes()
	return &responseBody, err
}

// The response object contains response information from the YARPC handler
type response struct {
	body    bytes.Buffer
	headers metadata.MD
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
	rw.r.headers = applicationHeaders.ToGRPCMetadata(h, rw.r.headers)
}

func (responseWriter) SetApplicationError() {
	// Nothing to do.
}
