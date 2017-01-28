package hashring

import (
	"github.com/yarpc/yab/transport"
	"go.uber.org/yarpc/api/peer"
	"golang.org/x/net/context"
)

func (*HashRing) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	return nil, nil, nil
}
