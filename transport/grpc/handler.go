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
	"google.golang.org/grpc/metadata"
	gtransport "google.golang.org/grpc/transport"
)

var grpcOptions transport.Options

type handler struct {
	Registry transport.Registry
	Deps     transport.Deps
}

// Handle the grpc request and convert it into a YARPC request
// dec ('decode') will pass through the request body in raw bytes using the passThroughCodec
func (h handler) handle(
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
	return callHandler(ctx, h, start, treq)
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

	appHeaders := applicationHeaders.fromMetadata(ctxMetadata, transport.Headers{})

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

func getContextMetadata(ctx context.Context) (metadata.MD, error) {
	contextMetadata, ok := metadata.FromContext(ctx)
	if !ok || contextMetadata == nil {
		return nil, errCantExtractMetadata{ctx: ctx}
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
	headerList, _ := ctxMetadata[header]
	if len(headerList) != 1 {
		return "", errCantExtractHeader{Name: header, MD: ctxMetadata}
	}
	return headerList[0], nil
}

func getMsgBody(msgBodyDecoder func(interface{}) error) (io.Reader, error) {
	var requestBody []byte
	if err := msgBodyDecoder(&requestBody); err != nil {
		return nil, err
	}
	return bytes.NewBuffer(requestBody), nil
}

func callHandler(
	ctx context.Context,
	h handler,
	start time.Time,
	treq *transport.Request,
) (interface{}, error) {
	var r response
	rw := newResponseWriter(&r)

	handler, err := h.Registry.GetHandler(treq.Service, treq.Procedure)
	if err != nil {
		return nil, err
	}

	err = internal.SafelyCallHandler(ctx, handler, start, grpcOptions, treq, rw)

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
	rw.r.headers = applicationHeaders.toMetadata(h, rw.r.headers)
}

func (responseWriter) SetApplicationError() {
	// Nothing to do.
}
