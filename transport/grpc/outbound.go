package grpc

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/yarpc/yarpc-go/transport"

	"net/url"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
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
	conn, err := grpc.Dial(o.address, grpc.WithInsecure(), grpc.WithCodec(PassThroughCodec{}))
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
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("/%s/%s", url.QueryEscape(req.Service), url.QueryEscape(req.Procedure))

	return callDownstream(ctx, uri, &requestBody, o.conn)
}

func callDownstream(
	ctx context.Context,
	uri string,
	requestBody *[]byte,
	connection *grpc.ClientConn,
) (*transport.Response, error) {
	var responseBody []byte

	if err := grpc.Invoke(ctx, uri, requestBody, &responseBody, connection); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(responseBody)
	closer := ioutil.NopCloser(buf)

	return &transport.Response{Body: closer}, nil
}
