package grpc

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/yarpc/yarpc-go/transport"

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

func (o *outbound) Start() error {
	conn, err := grpc.Dial(o.address, grpc.WithInsecure(), grpc.WithCodec(RawCodec{}))
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
	r, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	rr := string(r)

	uri := fmt.Sprintf("/%s/%s", req.Service, req.Procedure)
	var res string
	if err := grpc.Invoke(ctx, uri, &rr, &res, o.conn); err != nil {
		return nil, err
	}
	buf := bytes.NewBufferString(res)
	closer := ioutil.NopCloser(buf)

	return &transport.Response{Body: closer}, nil
}
