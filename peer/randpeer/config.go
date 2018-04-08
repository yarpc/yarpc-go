package randpeer

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/yarpcconfig"
)

// Spec returns a configuration specification for the random peer list
// implementation, making it possible to select a random peer with transports
// that use outbound peer list configuration (like HTTP).
//
//  cfg := yarpcconfig.New()
//  cfg.MustRegisterPeerList(random.Spec())
//
// This enables the random peer list:
//
//  outbounds:
//    otherservice:
//      unary:
//        http:
//          url: https://host:port/rpc
//          random:
//            peers:
//              - 127.0.0.1:8080
//              - 127.0.0.1:8081
func Spec() yarpcconfig.PeerListSpec {
	return yarpcconfig.PeerListSpec{
		Name: "random",
		BuildPeerList: func(c struct{}, t peer.Transport, k *yarpcconfig.Kit) (peer.ChooserList, error) {
			return New(t), nil
		},
	}
}
