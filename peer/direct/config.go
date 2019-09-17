package direct

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/yarpcconfig"
)

const name = "direct"

// Configuration describes how to build a direct peer chooser.
type Configuration struct{}

// Spec returns a configuration specification for the direct peer chooser. The
// chooser uses transport.Request#ShardKey as the peer dentifier.
//
//  cfg := yarpcconfig.New()
//  cfg.MustRegisterPeerChooser(direct.Spec())
//
// This enables the direct chooser:
//
//  outbounds:
//    destination-service:
//      grpc:
//        direct: {}
func Spec() yarpcconfig.PeerChooserSpec {
	return yarpcconfig.PeerChooserSpec{
		Name: name,
		BuildPeerChooser: func(cfg Configuration, t peer.Transport, _ *yarpcconfig.Kit) (peer.Chooser, error) {
			return New(cfg, t)
		},
	}
}
