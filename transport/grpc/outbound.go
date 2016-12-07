package grpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// NewOutbound builds a new GRPC outbound.
func NewOutbound(address string) transport.Outbound {
	return &outbound{address: address}
}

type outbound struct {
	address string
	conn    *grpc.ClientConn
}

func (o *outbound) Start(d transport.Deps) error {
	conn, err := grpc.Dial(o.address, grpc.WithInsecure(), grpc.WithCodec(passThroughCodec{}))
	if err != nil {
		return err
	}
	o.conn = conn
	return nil
}

func (o *outbound) Stop() error {
	o.conn.Close()
	return nil
}

func (outbound) Options() (o transport.Options) {
	return o
}

func (o outbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	start := time.Now()
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)

	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	metadataHeaders := getRequestHeaders(ctx, req)
	ctx = metadata.NewContext(ctx, metadataHeaders)
	uri := fmt.Sprintf("/%s/%s", url.QueryEscape(req.Service), url.QueryEscape(req.Procedure))

	response, err := callDownstream(ctx, uri, &requestBody, o.conn)
	return response, getErrFromGRPCError(err, req, ttl)
}

func getRequestHeaders(ctx context.Context, req *transport.Request) metadata.MD {
	// 'Headers' in gRPC are known as 'Metadata'
	md := metadata.New(map[string]string{
		CallerHeader:   req.Caller,
		EncodingHeader: string(req.Encoding),
	})
	return applicationHeaders.toMetadata(req.Headers, md)
}

func callDownstream(
	ctx context.Context,
	uri string,
	requestBody *[]byte,
	connection *grpc.ClientConn,
) (*transport.Response, error) {
	var responseBody []byte
	var responseHeaders metadata.MD

	if err := grpc.Invoke(ctx, uri, requestBody, &responseBody, connection, grpc.Header(&responseHeaders)); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(responseBody)
	closer := ioutil.NopCloser(buf)
	headers := applicationHeaders.fromMetadata(responseHeaders, transport.Headers{})
	return &transport.Response{Body: closer, Headers: headers}, nil
}

func getErrFromGRPCError(err error, treq *transport.Request, ttl time.Duration) error {
	if err == nil {
		return nil
	}

	switch grpc.Code(err) {
	// TIMEOUT
	case codes.DeadlineExceeded:
		return errors.ClientTimeoutError(treq.Service, treq.Procedure, ttl)

	// BAD REQUEST
	case codes.Unimplemented, codes.InvalidArgument, codes.NotFound:
		return errors.RemoteBadRequestError(grpc.ErrorDesc(err))

	// UNEXPECTED
	case codes.Canceled, codes.AlreadyExists, codes.PermissionDenied,
		codes.Unauthenticated, codes.ResourceExhausted, codes.FailedPrecondition,
		codes.Aborted, codes.OutOfRange, codes.Internal,
		codes.Unavailable, codes.DataLoss, codes.Unknown:
		fallthrough
	default:
		return errors.RemoteUnexpectedError(grpc.ErrorDesc(err))
	}
}
