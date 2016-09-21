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
	r, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("/%s/%s", req.Service, req.Procedure)
	var res []byte
	if err := grpc.Invoke(ctx, uri, &r, &res, o.conn); err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(res)
	closer := ioutil.NopCloser(buf)

	return &transport.Response{Body: closer}, nil
}
